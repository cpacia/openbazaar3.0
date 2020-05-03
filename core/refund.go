package core

import (
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

// RefundOrder sends a REFUND message to the remote peer and updates the node's
// order state. Only a vendor can call this method and only if the order has been opened
// and has received at least one payment.
//
// When this method is called we will refund the total amount received into the payment
// address (and not yet refunded). Note that this method can be called more than once.
// If new transactions were received in the payment address after a prior refund was
// sent, the remaining balance (and only the remaining balance) will be refunded.
func (n *OpenBazaarNode) RefundOrder(orderID models.OrderID, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).First(&order).Error
	})
	if err != nil {
		return err
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return err
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	if !order.CanRefund(n.Identity()) {
		return errors.New("order is not in a state where it can be refunded ")
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		wallet, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
		if err != nil {
			return err
		}
		wTx, refundMsg, err := n.buildRefundMessage(&order, wallet)
		if err != nil {
			return err
		}

		if err := utils.SignOrderMessage(refundMsg, n.ipfsNode.PrivateKey); err != nil {
			return err
		}

		refundPayload, err := ptypes.MarshalAny(refundMsg)
		if err != nil {
			wTx.Rollback()
			return err
		}

		message := newMessageWithID()
		message.MessageType = npb.Message_ORDER
		message.Payload = refundPayload

		_, err = n.orderProcessor.ProcessMessage(tx, vendor, refundMsg)
		if err != nil {
			wTx.Rollback()
			return err
		}
		if err := n.messenger.ReliablySendMessage(tx, buyer, message, done); err != nil {
			wTx.Rollback()
			return err
		}

		return wTx.Commit()
	})
}

func (n *OpenBazaarNode) buildRefundMessage(order *models.Order, wallet iwallet.Wallet) (iwallet.Tx, *npb.OrderMessage, error) {
	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return nil, nil, err
	}
	var (
		refundAddress   = iwallet.NewAddress(orderOpen.RefundAddress, iwallet.CoinType(orderOpen.Payment.Coin))
		refundMsg       = newMessageWithID()
		prevRefundTotal = iwallet.NewAmount(0)
		refundPayload   *any.Any
		refundResp      = &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_REFUND,
		}
	)

	wdbTx, err := wallet.Begin()
	if err != nil {
		return nil, nil, err
	}

	switch orderOpen.Payment.Method {
	case pb.OrderOpen_Payment_DIRECT:
		fundingTotal, err := order.FundingTotal()
		if err != nil {
			return nil, nil, err
		}
		previousRefunds, _ := order.Refunds()
		for _, refund := range previousRefunds {
			prevRefundTotal = prevRefundTotal.Add(iwallet.NewAmount(refund.Amount))
		}

		refundTotal := fundingTotal.Sub(prevRefundTotal)

		txid, err := wallet.Spend(wdbTx, refundAddress, refundTotal, iwallet.FlNormal)
		if err != nil {
			return nil, nil, err
		}

		refund := pb.Refund{
			RefundInfo: &pb.Refund_TransactionID{TransactionID: txid.String()},
			Amount:     refundTotal.String(),
			Timestamp:  ptypes.TimestampNow(),
		}

		refundAny, err := ptypes.MarshalAny(&refund)
		if err != nil {
			return nil, nil, err
		}

		refundResp.Message = refundAny
	case pb.OrderOpen_Payment_MODERATED:
		release, err := n.buildEscrowRelease(order, wallet,
			iwallet.NewAddress(orderOpen.RefundAddress, iwallet.CoinType(orderOpen.Payment.Coin)),
			iwallet.NewAmount(orderOpen.Payment.EscrowReleaseFee))
		if err != nil {
			return nil, nil, err
		}
		refund := pb.Refund{
			RefundInfo: &pb.Refund_ReleaseInfo{
				ReleaseInfo: release,
			},
			Amount: release.ToAmount,
		}

		refundAny, err := ptypes.MarshalAny(&refund)
		if err != nil {
			return nil, nil, err
		}

		refundResp.Message = refundAny
	default:
		return nil, nil, errors.New("unknown payment method")
	}

	refundMsg.MessageType = npb.Message_ORDER
	refundMsg.Payload = refundPayload
	return wdbTx, refundResp, nil
}

func (n *OpenBazaarNode) buildEscrowRelease(order *models.Order, wallet iwallet.Wallet, to iwallet.Address, escrowReleaseFee iwallet.Amount) (*pb.EscrowRelease, error) {
	escrowWallet, ok := wallet.(iwallet.Escrow)
	if !ok {
		return nil, errors.New("wallet does not support escrow")
	}

	txs, err := order.GetTransactions()
	if err != nil {
		return nil, err
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return nil, err
	}

	var (
		txn      iwallet.Transaction
		totalOut = iwallet.NewAmount(0)
	)
	spent := make(map[string]bool)
	for _, tx := range txs {
		for _, from := range tx.From {
			spent[hex.EncodeToString(from.ID)] = true
		}
	}
	for _, tx := range txs {
		for _, to := range tx.To {
			if !spent[hex.EncodeToString(to.ID)] && to.Address.String() == orderOpen.Payment.Address {
				txn.From = append(txn.From, to)
				totalOut = totalOut.Add(to.Amount)
			}
		}
	}
	txn.To = append(txn.To, iwallet.SpendInfo{
		Address: to,
		Amount:  totalOut.Sub(escrowReleaseFee),
	})

	script, err := hex.DecodeString(orderOpen.Payment.Script)
	if err != nil {
		return nil, err
	}

	chainCode, err := hex.DecodeString(orderOpen.Payment.Chaincode)
	if err != nil {
		return nil, err
	}

	vendorKey, err := utils.GenerateEscrowPrivateKey(n.escrowMasterKey, chainCode)
	if err != nil {
		return nil, err
	}

	sigs, err := escrowWallet.SignMultisigTransaction(txn, *vendorKey, script)
	if err != nil {
		return nil, err
	}

	release := &pb.EscrowRelease{
		ToAddress:      txn.To[0].Address.String(),
		ToAmount:       txn.To[0].Amount.String(),
		TransactionFee: escrowReleaseFee.String(),
	}

	for _, from := range txn.From {
		release.FromIDs = append(release.FromIDs, from.ID)
	}

	for _, sig := range sigs {
		release.EscrowSignatures = append(release.EscrowSignatures, &pb.Signature{
			Signature: sig.Signature,
			Index:     uint32(sig.Index),
		})
	}

	return release, nil
}

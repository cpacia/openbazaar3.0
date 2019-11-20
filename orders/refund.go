package orders

import (
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	"math/big"
)

func (op *OrderProcessor) processRefundMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	refund := new(pb.Refund)
	if err := ptypes.UnmarshalAny(message.Message, refund); err != nil {
		return nil, err
	}

	if order.SerializedOrderCancel != nil {
		log.Errorf("Received REFUND message for order %s after ORDER_CANCEL", order.ID)
		return nil, ErrUnexpectedMessage
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	if err := order.PutMessage(refund); err != nil {
		if models.IsDuplicateTransactionError(err) {
			return nil, nil
		}
		return nil, err
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	if refund.GetTransactionID() != "" && orderOpen.Payment.Method == pb.OrderOpen_Payment_DIRECT {
		// If this fails it's OK as the processor's unfunded order checking loop will
		// retry at it's next interval.
		tx, err := wallet.GetTransaction(iwallet.TransactionID(refund.GetTransactionID()))
		if err == nil {
			for _, from := range tx.From {
				if from.Address.String() == order.PaymentAddress {
					if err := op.processOutgoingPayment(dbtx, order, tx); err != nil {
						return nil, err
					}
				}
			}
		}
	} else if order.Role() == models.RoleBuyer && refund.GetReleaseInfo() != nil && orderOpen.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		if err := op.releaseEscrowFunds(wallet, orderOpen, refund.GetReleaseInfo()); err != nil {
			log.Errorf("Error releasing funds from escrow during refund processing: %s", err.Error())
		}
	}

	if order.Role() == models.RoleBuyer {
		log.Infof("Received REFUND message for order %s", order.ID)
	} else if order.Role() == models.RoleVendor {
		log.Infof("Processed own REFUND for order %s", order.ID)
	}

	event := &events.RefundNotification{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}
	return event, nil
}

func (op *OrderProcessor) releaseEscrowFunds(wallet iwallet.Wallet, orderOpen *pb.OrderOpen, releaseInfo *pb.Refund_EscrowRelease) error {
	escrowWallet, ok := wallet.(iwallet.Escrow)
	if !ok {
		return errors.New("wallet for moderated order does not support escrow")
	}

	if releaseInfo.ToAddress != orderOpen.RefundAddress {
		return errors.New("Refund does not pay out to our refund address")
	}
	_, ok = new(big.Int).SetString(releaseInfo.ToAmount, 10)
	if !ok {
		return errors.New("Invalid payment amount")
	}
	txn := iwallet.Transaction{
		To: []iwallet.SpendInfo{
			{
				Address: iwallet.NewAddress(releaseInfo.ToAddress, iwallet.CoinType(orderOpen.Payment.Coin)),
				Amount:  iwallet.NewAmount(releaseInfo.ToAmount),
			},
		},
	}

	for _, id := range releaseInfo.FromIDs {
		txn.From = append(txn.From, iwallet.SpendInfo{ID: id})
	}

	var vendorSigs []iwallet.EscrowSignature
	for _, sig := range releaseInfo.EscrowSignatures {
		vendorSigs = append(vendorSigs, iwallet.EscrowSignature{
			Index:     int(sig.Index),
			Signature: sig.Signature,
		})
	}

	script, err := hex.DecodeString(orderOpen.Payment.Script)
	if err != nil {
		return err
	}

	chainCode, err := hex.DecodeString(orderOpen.Payment.Chaincode)
	if err != nil {
		return err
	}

	buyerKey, err := utils.GenerateEscrowPrivateKey(op.escrowPrivateKey, chainCode)
	if err != nil {
		return err
	}

	buyerSigs, err := escrowWallet.SignMultisigTransaction(txn, *buyerKey, script)
	if err != nil {
		return err
	}
	dbtx, err := wallet.Begin()
	if err != nil {
		return err
	}
	if _, err := escrowWallet.BuildAndSend(dbtx, txn, [][]iwallet.EscrowSignature{buyerSigs, vendorSigs}, script); err != nil {
		return err
	}

	return dbtx.Commit()
}

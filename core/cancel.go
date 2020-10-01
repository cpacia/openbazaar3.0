package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
)

// CancelOrder is called only by the buyer and sends an ORDER_CANCEL message to the vendor
// while releasing the funds from the 1 of 2 multisig back into this wallet. This can only
// be called when the order payment method is CANCELABLE and the order has not been confirmed
// or progressed any further.
//
// Note there is a possibility of a race between this function and ConfirmOrder called by
// the vendor. In such a scenario this function will return without error but we will
// later determine which person "wins" based on which transaction confirmed in the blockchain.
func (n *OpenBazaarNode) CancelOrder(orderID models.OrderID, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).First(&order).Error
	})
	if err != nil {
		return err
	}

	if !order.CanCancel() {
		return fmt.Errorf("%w: order is not in a state where it can be canceled", coreiface.ErrBadRequest)
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	wTx, txid, err := n.releaseFromCancelableAddress(&order)
	if err != nil {
		return err
	}

	cancel := &pb.OrderCancel{
		TransactionID: txid.String(),
		Timestamp:     ptypes.TimestampNow(),
	}

	cancelAny, err := ptypes.MarshalAny(cancel)
	if err != nil {
		return err
	}

	resp := &npb.OrderMessage{
		OrderID:     order.ID.String(),
		MessageType: npb.OrderMessage_ORDER_CANCEL,
		Message:     cancelAny,
	}

	if err := utils.SignOrderMessage(resp, n.ipfsNode.PrivateKey); err != nil {
		return err
	}

	payload, err := ptypes.MarshalAny(resp)
	if err != nil {
		return err
	}

	message := newMessageWithID()
	message.MessageType = npb.Message_ORDER
	message.Payload = payload

	return n.repo.DB().Update(func(tx database.Tx) error {
		_, err = n.orderProcessor.ProcessMessage(tx, buyer, resp)
		if err != nil {
			wTx.Rollback()
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, vendor, message, done); err != nil {
			wTx.Rollback()
			return err
		}

		return wTx.Commit()
	})
}

func (n *OpenBazaarNode) releaseFromCancelableAddress(order *models.Order) (iwallet.Tx, iwallet.TransactionID, error) {
	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return nil, "", err
	}

	if orderOpen.Payment.Method != pb.OrderOpen_Payment_CANCELABLE {
		return nil, "", errors.New("order payment method is not CANCELABLE")
	}

	wallet, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, "", err
	}

	toAddress, err := wallet.CurrentAddress()
	if err != nil {
		return nil, "", err
	}

	escrowWallet, ok := wallet.(iwallet.Escrow)
	if !ok {
		return nil, "", errors.New("wallet does not support escrow")
	}

	txs, err := order.GetTransactions()
	if err != nil {
		return nil, "", err
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

	if len(txn.From) == 0 {
		return nil, "", errors.New("payment address is empty")
	}

	escrowFee, err := escrowWallet.EstimateEscrowFee(1, iwallet.FlNormal)
	if err != nil {
		return nil, "", err
	}
	// The escrow fee is calculated as 100% of EstimateEscrowFee for the first input.
	// Plus 50% of EstimateEscrowFee for each additional input.
	escrowFee = escrowFee.Add(escrowFee.Div(iwallet.NewAmount(2)).Mul(iwallet.NewAmount(len(txn.From) - 1)))

	txn.To = append(txn.To, iwallet.SpendInfo{
		Address: toAddress,
		Amount:  totalOut.Sub(escrowFee),
	})

	script, err := hex.DecodeString(orderOpen.Payment.Script)
	if err != nil {
		return nil, "", err
	}

	chainCode, err := hex.DecodeString(orderOpen.Payment.Chaincode)
	if err != nil {
		return nil, "", err
	}

	key, err := utils.GenerateEscrowPrivateKey(n.escrowMasterKey, chainCode)
	if err != nil {
		return nil, "", err
	}

	sigs, err := escrowWallet.SignMultisigTransaction(txn, *key, script)
	if err != nil {
		return nil, "", err
	}

	dbTx, err := wallet.Begin()
	if err != nil {
		return nil, "", err
	}

	txid, err := escrowWallet.BuildAndSend(dbTx, txn, [][]iwallet.EscrowSignature{sigs}, script)
	if err != nil {
		return nil, "", err
	}
	return dbTx, txid, nil
}

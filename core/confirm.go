package core

import (
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

// ConfirmOrder sends a ORDER_CONFIRMATION message to the remote peer and updates the node's
// order state. Only a vendor can call this method and only if the order has been opened
// and no other actions have been taken.
//
// If the payment method is CANCELABLE, this will attempt to move the funds into the vendor's
// wallet. Note that there is a potential for a race between this function being called by
// the vendor and CancelOrder being called by the buyer. In such a scenario this function
// will return without error, however we determine which person "wins" based on which transaction
// confirms in the blockchain.
func (n *OpenBazaarNode) ConfirmOrder(orderID models.OrderID, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).First(&order).Error
	})
	if err != nil {
		return err
	}

	if !order.CanConfirm(n.Identity()) {
		return fmt.Errorf("%w: order is not in a state where it can be confirmed", coreiface.ErrBadRequest)
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		orderOpen, err := order.OrderOpenMessage()
		if err != nil {
			return err
		}

		var (
			wTx          iwallet.Tx
			txid         iwallet.TransactionID
			confirmation = &pb.OrderConfirmation{
				Timestamp: ptypes.TimestampNow(),
			}
		)
		if orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
			wTx, txid, err = n.releaseFromCancelableAddress(&order)
			if err != nil {
				return err
			}
			confirmation.TransactionID = txid.String()
		}

		confirmAny, err := ptypes.MarshalAny(confirmation)
		if err != nil {
			return err
		}

		resp := &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_ORDER_CONFIRMATION,
			Message:     confirmAny,
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

		_, err = n.orderProcessor.ProcessMessage(tx, vendor, resp)
		if err != nil {
			if orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
				wTx.Rollback()
			}
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, buyer, message, done); err != nil {
			if orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
				wTx.Rollback()
			}
			return err
		}

		if orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
			return wTx.Commit()
		}
		return nil
	})
}

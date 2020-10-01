package core

import (
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/ptypes"
)

// RejectOrder sends a ORDER_REJECT message to the remote peer and updates the node's
// order state. Only a vendor can call this method and only if the order has been opened
// and no other actions have been taken.
func (n *OpenBazaarNode) RejectOrder(orderID models.OrderID, reason string, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).First(&order).Error
	})
	if err != nil {
		return err
	}

	if !order.CanReject() {
		return fmt.Errorf("%w: order is not in a state where it can be rejected", coreiface.ErrBadRequest)
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	reject := pb.OrderReject{
		Type:      pb.OrderReject_USER_REJECT,
		Reason:    reason,
		Timestamp: ptypes.TimestampNow(),
	}

	rejectAny, err := ptypes.MarshalAny(&reject)
	if err != nil {
		return err
	}

	resp := npb.OrderMessage{
		OrderID:     order.ID.String(),
		MessageType: npb.OrderMessage_ORDER_REJECT,
		Message:     rejectAny,
	}

	if err := utils.SignOrderMessage(&resp, n.ipfsNode.PrivateKey); err != nil {
		return err
	}

	payload, err := ptypes.MarshalAny(&resp)
	if err != nil {
		return err
	}

	message := newMessageWithID()
	message.MessageType = npb.Message_ORDER
	message.Payload = payload

	funded, err := order.IsFunded()
	if err != nil {
		return err
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return err
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		_, err := n.orderProcessor.ProcessMessage(tx, vendor, &resp)
		if err != nil {
			return err
		}

		// If the order is funded and not a CANCELABLE order we need to send the refund as well.
		if funded && orderOpen.Payment.Method != pb.OrderOpen_Payment_CANCELABLE {
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

			refundResp := newMessageWithID()
			refundResp.MessageType = npb.Message_ORDER
			refundResp.Payload = refundPayload

			_, err = n.orderProcessor.ProcessMessage(tx, vendor, refundMsg)
			if err != nil {
				wTx.Rollback()
				return err
			}

			var (
				done1 = make(chan struct{})
				done2 = make(chan struct{})
			)

			if err := n.messenger.ReliablySendMessage(tx, buyer, message, done1); err != nil {
				wTx.Rollback()
				return err
			}

			if err := n.messenger.ReliablySendMessage(tx, buyer, refundResp, done2); err != nil {
				wTx.Rollback()
				return err
			}

			go func() {
				<-done1
				<-done2
				close(done)
			}()

			return wTx.Commit()
		}

		return n.messenger.ReliablySendMessage(tx, buyer, message, done)
	})
}

package core

import (
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (n *OpenBazaarNode) OpenDispute(orderID models.OrderID, reason string, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Find(&order).Error
	})
	if err != nil {
		return err
	}

	// FIXME: check can dispute here

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

	var (
		role = pb.DisputeOpen_BUYER
		to   = vendor
		from = buyer
	)
	if order.Role() == models.RoleVendor {
		role = pb.DisputeOpen_VENDOR
		to = buyer
		from = vendor
	}

	serializedContract, err := order.MarshalJSON()
	if err != nil {
		return err
	}

	disputeOpen := &pb.DisputeOpen{
		Timestamp: ptypes.TimestampNow(),
		OpenedBy:  role,
		Reason:    reason,
		Contract:  serializedContract,
	}

	moderatorID, err := peer.Decode(orderOpen.Payment.Moderator)
	if err != nil {
		return err
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		disputeOpenAny, err := ptypes.MarshalAny(disputeOpen)
		if err != nil {
			return err
		}

		m := &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_DISPUTE_OPEN,
			Message:     disputeOpenAny,
		}

		if err := utils.SignOrderMessage(m, n.ipfsNode.PrivateKey); err != nil {
			return err
		}

		payload, err := ptypes.MarshalAny(m)
		if err != nil {
			return err
		}

		message1 := newMessageWithID()
		message1.MessageType = npb.Message_ORDER
		message1.Payload = payload

		_, err = n.orderProcessor.ProcessMessage(tx, from, m)
		if err != nil {
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, to, message1, done); err != nil {
			return err
		}

		message2 := newMessageWithID()
		message2.MessageType = npb.Message_DISPUTE
		message2.Payload = payload

		if err := n.messenger.ReliablySendMessage(tx, moderatorID, message2, done); err != nil {
			return err
		}

		return nil
	})
}

// handleOrderMessage is the handler for the ORDER message. It sends it off to the order
// order processor for processing.
func (n *OpenBazaarNode) handleDisputeMessage(from peer.ID, message *npb.Message) error {
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	if message.MessageType != npb.Message_DISPUTE {
		return errors.New("message is not type DISPUTE")
	}

	order := new(npb.OrderMessage)
	if err := ptypes.UnmarshalAny(message.Payload, order); err != nil {
		return err
	}

	switch order.MessageType {
	case npb.OrderMessage_DISPUTE_OPEN:
		disputeOpen := new(pb.DisputeOpen)
		if err := ptypes.UnmarshalAny(order.Message, disputeOpen); err != nil {
			return err
		}

		// TODO: validate dispute open

		return n.repo.DB().Update(func(dbtx database.Tx) error {
			var disputeCase models.Case
			err := dbtx.Read().Where("id = ?", order.OrderID).First(&disputeCase).Error
			if err != nil && !gorm.IsRecordNotFoundError(err) {
				return err
			}

			if !gorm.IsRecordNotFoundError(err) {
				return fmt.Errorf("duplicate DISPUTE_OPEN for order %s", order.OrderID)
			}

			return disputeCase.PutDisputeOpen(disputeOpen)
		})
	case npb.OrderMessage_DISPUTE_UPDATE:

	}
	return nil
}

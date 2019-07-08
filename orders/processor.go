package orders

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("ORDR")

type OrderProcessor struct {
	db          database.Database
	messenger   *net.Messenger
	multiwallet wallet.Multiwallet
}

// NewOrderProcessor initializes and returns a new OrderProcessor
func NewOrderProcessor(db database.Database, messenger *net.Messenger, multiwallet wallet.Multiwallet) *OrderProcessor {
	return &OrderProcessor{db, messenger, multiwallet}
}

func (op *OrderProcessor) ProcessMessage(peer peer.ID, message *npb.OrderMessage) error {
	return op.db.Update(func(tx database.Tx) error {
		// Load the order if it exists.
		var order models.Order
		err := tx.DB().Where("order_id = ?", message.OrderID).First(&order).Error
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		} else if gorm.IsRecordNotFoundError(err) && message.MessageType != npb.OrderMessage_ORDER_OPEN {
			// Order does not exist in the DB and the message type is not an order open. This can happen
			// in the case where we download offline messages out of order. In this case we will park
			// the message so that we can try again later if we receive other messages.
			log.Warningf("Received %s message from peer %d for an order that does not exist yet", message.MessageType, peer)
			if err := order.ParkMessage(message); err != nil {
				return err
			}
			if err := tx.DB().Save(&order).Error; err != nil {
				return err
			}
			return nil
		}

		switch message.MessageType {
		case npb.OrderMessage_ORDER_OPEN:
			err = op.handleOrderOpenMessage(&order, message)
		default:
			return errors.New("unknown order message type")
		}

		// Save changes to the database.
		if err := tx.DB().Save(&order).Error; err != nil {
			return err
		}

		return err
	})
}

// ProcessACK loads the order from the database and sets the ACK for the message type.
func (op *OrderProcessor) ProcessACK(tx database.Tx, om *models.OutgoingMessage) error {
	message := new(npb.Message)
	if err := proto.Unmarshal(om.SerializedMessage, message); err != nil {
		return err
	}

	orderMessage := new(npb.OrderMessage)
	if err := ptypes.UnmarshalAny(message.Payload, orderMessage); err != nil {
		return err
	}

	dbtx := tx.DB().Where("order_id = ?", orderMessage.OrderID)

	switch orderMessage.MessageType {
	case npb.OrderMessage_ORDER_OPEN:
		return dbtx.Update("order_open_acked", true).Error
	case npb.OrderMessage_ORDER_REJECT:
		return dbtx.Update("order_reject_acked", true).Error
	case npb.OrderMessage_ORDER_CANCEL:
		return dbtx.Update("order_cancel_acked", true).Error
	case npb.OrderMessage_ORDER_CONFIRMATION:
		return dbtx.Update("order_confirmation_acked", true).Error
	case npb.OrderMessage_ORDER_FULFILLMENT:
		return dbtx.Update("order_fulfillment_acked", true).Error
	case npb.OrderMessage_ORDER_COMPLETE:
		return dbtx.Update("order_complete_acked", true).Error
	case npb.OrderMessage_DISPUTE_OPEN:
		return dbtx.Update("dispute_open_acked", true).Error
	case npb.OrderMessage_DISPUTE_UPDATE:
		return dbtx.Update("dispute_update_acked", true).Error
	case npb.OrderMessage_DISPUTE_CLOSE:
		return dbtx.Update("dispute_closed_acked", true).Error
	case npb.OrderMessage_REFUND:
		return dbtx.Update("refund_acked", true).Error
	case npb.OrderMessage_PAYMENT_SENT:
		return dbtx.Update("payment_sent_acked", true).Error
	case npb.OrderMessage_PAYMENT_FINALIZED:
		return dbtx.Update("payment_finalized_acked", true).Error
	default:
		return errors.New("unknown order message type")
	}
}

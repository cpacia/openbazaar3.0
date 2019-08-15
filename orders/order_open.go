package orders

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-peer"
)

func (op *OrderProcessor) handleOrderOpenMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	dup, err := isDuplicate(message, order.SerializedOrderOpen)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderOpen != nil && !dup {
		log.Error("Duplicate ORDER_OPEN message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	orderOpen := new(pb.OrderOpen)
	if err := ptypes.UnmarshalAny(message.Message, orderOpen); err != nil {
		return nil, err
	}

	var validationError bool
	// If the validation fails and we are the vendor, we send a REJECT message back
	// to the buyer. The reject message also gets saved with this order.
	if err := op.validateOrderOpen(dbtx, orderOpen); err != nil {
		log.Error("ORDER_OPEN message for order %s from %s failed to validate: %s", order.ID, orderOpen.BuyerID.PeerID, err)
		if op.identity != peer {
			reject := pb.OrderReject{
				Reason: err.Error(),
			}

			rejectAny, err := ptypes.MarshalAny(&reject)
			if err != nil {
				return nil, err
			}

			resp := npb.OrderMessage{
				OrderID:     order.ID.String(),
				MessageType: npb.OrderMessage_ORDER_REJECT,
				Message:     rejectAny,
			}

			payload, err := ptypes.MarshalAny(&resp)
			if err != nil {
				return nil, err
			}

			messageID := make([]byte, 20)
			if _, err := rand.Read(messageID); err != nil {
				return nil, err
			}

			message := npb.Message{
				MessageType: npb.Message_ORDER,
				MessageID:   hex.EncodeToString(messageID),
				Payload:     payload,
			}

			if err := op.messenger.ReliablySendMessage(dbtx, peer, &message, nil); err != nil {
				return nil, err
			}

			if err := order.PutMessage(&resp); err != nil {
				return nil, err
			}
		}
		validationError = true
	}

	var event interface{}
	// TODO: do we want to emit an event in the case of a validation error?
	if !validationError && op.identity != peer {
		event = &events.OrderNotification{
			ID: order.ID.String(),
		}
	}

	if err := order.PutMessage(orderOpen); err != nil {
		return nil, err
	}

	return event, nil
}

// CalculateOrderTotal calculates and returns the total for the order with all
// the provided options.
func CalculateOrderTotal(order *pb.OrderOpen) (iwallet.Amount, error) {
	// TODO
	return iwallet.NewAmount(0), nil
}

func (op *OrderProcessor) validateOrderOpen(dbtx database.Tx, order *pb.OrderOpen) error {
	// TODO

	// Check to make sure we actually have the item for sale.
	if op.identity.Pretty() != order.BuyerID.PeerID {
		index, err := dbtx.GetListingIndex()
		if err != nil {
			return err
		}
		for _, item := range order.Items {
			c, err := cid.Decode(item.ListingHash)
			if err != nil {
				return err
			}
			_, err = index.GetListingSlug(c)
			if err != nil {
				return fmt.Errorf("item %s is not for sale", item.ListingHash)
			}
		}
	}
	return nil
}
package orders

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
)

func (op *OrderProcessor) processOrderCancelMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	orderCancel := new(pb.OrderCancel)
	if err := ptypes.UnmarshalAny(message.Message, orderCancel); err != nil {
		return nil, err
	}
	dup, err := isDuplicate(orderCancel, order.SerializedOrderCancel)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderCancel != nil && !dup {
		log.Errorf("Duplicate ORDER_CANCEL message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	if order.SerializedOrderReject != nil {
		log.Errorf("Received ORDER_CANCEL message for order %s after ORDER_REJECT", order.ID)
		return nil, ErrUnexpectedMessage
	}

	if order.SerializedOrderConfirmation != nil {
		log.Errorf("Received ORDER_CANCEL message for order %s after ORDER_CONFIRMATION", order.ID)
		return nil, ErrUnexpectedMessage
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	event := &events.OrderCancelNotification{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		BuyerHandle: orderOpen.BuyerID.Handle,
		BuyerID:     orderOpen.BuyerID.PeerID,
	}

	if order.Role() == models.RoleBuyer {
		log.Infof("Processed own ORDER_CANCEL for orderID: %s", order.ID)
	} else if order.Role() == models.RoleVendor {
		log.Infof("Received ORDER_CANCEL message for order %s", order.ID)
	}

	return event, order.PutMessage(orderCancel)
}

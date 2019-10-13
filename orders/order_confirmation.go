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

func (op *OrderProcessor) processOrderConfirmationMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	orderConfirmation := new(pb.OrderConfirmation)
	if err := ptypes.UnmarshalAny(message.Message, orderConfirmation); err != nil {
		return nil, err
	}
	dup, err := isDuplicate(orderConfirmation, order.SerializedOrderConfirmation)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderConfirmation != nil && !dup {
		log.Errorf("Duplicate ORDER_CONFIRMATION message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	if order.SerializedOrderReject != nil {
		log.Errorf("Received ORDER_CONFIRMATION message for order %s after ORDER_REJECT", order.ID)
		return nil, ErrUnexpectedMessage
	}

	if order.SerializedOrderCancel != nil {
		log.Errorf("Received ORDER_CONFIRMATION message for order %s after ORDER_CANCEL", order.ID)
		return nil, ErrUnexpectedMessage
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	event := &events.OrderConfirmationNotification{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}

	log.Infof("Received ORDER_CONFIRMATION message for order %s", order.ID)

	return event, order.PutMessage(orderConfirmation)
}

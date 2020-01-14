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

func (op *OrderProcessor) processOrderRejectMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	orderReject := new(pb.OrderReject)
	if err := ptypes.UnmarshalAny(message.Message, orderReject); err != nil {
		return nil, err
	}
	dup, err := isDuplicate(orderReject, order.SerializedOrderReject)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderReject != nil && !dup {
		log.Errorf("Duplicate ORDER_REJECT message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	if order.SerializedOrderConfirmation != nil {
		log.Errorf("Received ORDER_REJECT message for order %s after ORDER_CONFIRMATION", order.ID)
		return nil, ErrUnexpectedMessage
	}

	if order.SerializedOrderCancel != nil {
		log.Warningf("Possible race: Received ORDER_REJECT message for order %s after ORDER_CANCEL", order.ID)
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	event := &events.OrderDeclined{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}

	if order.Role() == models.RoleBuyer {
		log.Infof("Received ORDER_REJECT message for order %s", order.ID)
	} else if order.Role() == models.RoleVendor {
		log.Infof("Processed own ORDER_REJECT for orderID: %s", order.ID)
	}

	return event, order.PutMessage(message)
}

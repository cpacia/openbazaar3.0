package orders

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func (op *OrderProcessor) processOrderFulfillmentMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	orderFulfillment := new(pb.OrderFulfillment)
	if err := ptypes.UnmarshalAny(message.Message, orderFulfillment); err != nil {
		return nil, err
	}

	if order.SerializedOrderReject != nil {
		log.Errorf("Received ORDER_FULFILLMENT message for order %s after ORDER_REJECT", order.ID)
		return nil, ErrUnexpectedMessage
	}

	if order.SerializedOrderCancel != nil {
		log.Warningf("Possible race: Received ORDER_FULFILLMENT message for order %s after ORDER_CANCEL", order.ID)
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	_, err = order.OrderConfirmationMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	event := &events.OrderFulfillment{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}

	if order.Role() == models.RoleBuyer {
		log.Infof("Received ORDER_FULFILLMENT message for order %s", order.ID)
	} else if order.Role() == models.RoleVendor {
		log.Infof("Processed own ORDER_FULFILLMENT for order %s", order.ID)
	}

	return event, order.PutMessage(message)
}

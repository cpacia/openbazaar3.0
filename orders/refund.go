package orders

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
)

func (op *OrderProcessor) processRefundMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	refund := new(pb.Refund)
	if err := ptypes.UnmarshalAny(message.Message, refund); err != nil {
		return nil, err
	}

	dup, err := isDuplicate(refund, order.SerializedRefund)
	if err != nil {
		return nil, err
	}
	if order.SerializedRefund != nil && !dup {
		log.Errorf("Duplicate ORDER_REFUND message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	if err := order.PutMessage(refund); err != nil {
		return nil, err
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	// If this fails it's OK as the processor's unfunded order checking loop will
	// retry at it's next interval.
	tx, err := wallet.GetTransaction(iwallet.TransactionID(refund.TransactionID))
	if err == nil {
		for _, from := range tx.From {
			if from.Address.String() == order.PaymentAddress {
				if err := op.processOutgoingPayment(dbtx, order, tx); err != nil {
					return nil, err
				}
			}
		}
	}

	log.Infof("Received REFUND message for order %s", order.ID)

	event := &events.RefundNotification{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}
	return event, nil
}

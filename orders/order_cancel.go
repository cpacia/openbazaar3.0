package orders

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	crypto "github.com/libp2p/go-libp2p-crypto"
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
		log.Warningf("Possible race: Received ORDER_CANCEL message for order %s after ORDER_REJECT", order.ID)
	}

	if order.SerializedOrderConfirmation != nil {
		log.Warningf("Possible race: Received ORDER_CANCEL message for order %s after ORDER_CONFIRMATION", order.ID)
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	buyerPubkey, err := crypto.UnmarshalPublicKey(orderOpen.BuyerID.Pubkeys.Identity)
	if err != nil {
		return nil, err
	}

	valid, err := buyerPubkey.Verify([]byte(order.ID.String()), orderCancel.Signature)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, errors.New("invalid buyer signature on order cancel")
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	if orderCancel.TransactionID != "" && orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
		// If this fails it's OK as the processor's unfunded order checking loop will
		// retry at it's next interval.
		tx, err := wallet.GetTransaction(iwallet.TransactionID(orderCancel.TransactionID))
		if err == nil {
			log.Info("Processing tx")
			for _, from := range tx.From {
				if from.Address.String() == order.PaymentAddress {
					if err := op.processOutgoingPayment(dbtx, order, tx); err != nil {
						return nil, err
					}
				}
			}
		}
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

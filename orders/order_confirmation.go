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
		log.Warningf("Possible race: Received ORDER_CONFIRMATION message for order %s after ORDER_CANCEL", order.ID)
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	vendorPubkey, err := crypto.UnmarshalPublicKey(orderOpen.Listings[0].Listing.VendorID.Pubkeys.Identity)
	if err != nil {
		return nil, err
	}

	valid, err := vendorPubkey.Verify([]byte(order.ID.String()), orderConfirmation.Signature)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, errors.New("invalid vendor signature on order confirmation")
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	if orderConfirmation.TransactionID != "" && orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
		// If this fails it's OK as the processor's unfunded order checking loop will
		// retry at it's next interval.
		tx, err := wallet.GetTransaction(iwallet.TransactionID(orderConfirmation.TransactionID))
		if err == nil {
			for _, from := range tx.From {
				if from.Address.String() == order.PaymentAddress {
					if err := op.processOutgoingPayment(dbtx, order, tx); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	event := &events.OrderConfirmation{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		VendorHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		VendorID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}

	if order.Role() == models.RoleBuyer {
		log.Infof("Received ORDER_CONFIRMATION message for order %s", order.ID)
	} else if order.Role() == models.RoleVendor {
		log.Infof("Processed own ORDER_CONFIRMATION for order %s", order.ID)
	}

	return event, order.PutMessage(orderConfirmation)
}

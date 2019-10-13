package orders

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
)

// processWalletTransaction scan's through a transaction's inputs and outputs and attempts
// to load the order for that address from the database. If an order is found, the transaction
// is handed off to the appropriate handler for further processing.
func (op *OrderProcessor) processWalletTransaction(transaction iwallet.Transaction) {
	err := op.db.Update(func(tx database.Tx) error {
		for _, to := range transaction.To {
			var order models.Order
			err := tx.Read().Where("payment_address = ?", to.Address.String()).First(&order).Error
			if gorm.IsRecordNotFoundError(err) {
				continue
			} else if err != nil {
				return err
			}

			if err := op.processIncomingPayment(tx, &order, transaction); err != nil {
				return err
			}

			if err := tx.Save(&order); err != nil {
				return err
			}
		}
		for _, from := range transaction.From {
			var order models.Order
			err := tx.Read().Where("payment_address = ?", from.Address.String()).First(&order).Error
			if gorm.IsRecordNotFoundError(err) {
				continue
			} else if err != nil {
				return err
			}

			if err := op.processOutgoingPayment(tx, &order, transaction); err != nil {
				return err
			}

			if err := tx.Save(&order); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Errorf("Error handling incoming order transaction %s: %s", transaction.ID, err)
	}
}

// processIncomingPayment processes payments into an order's payment address.
func (op *OrderProcessor) processIncomingPayment(dbtx database.Tx, order *models.Order, tx iwallet.Transaction) error {
	err := order.PutTransaction(tx)
	if models.IsDuplicateTransactionError(err) {
		log.Debugf("Received duplicate transaction %s", tx.ID.String())
		return nil
	} else if err != nil {
		return err
	}

	funded, err := order.IsFunded()
	if err != nil {
		return err
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return err
	}

	switch order.Role() {
	case models.RoleBuyer:
		payment := pb.PaymentSent{
			TransactionID: tx.ID.String(),
		}

		paymentAny, err := ptypes.MarshalAny(&payment)
		if err != nil {
			return err
		}

		resp := npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_PAYMENT_SENT,
			Message:     paymentAny,
		}

		payload, err := ptypes.MarshalAny(&resp)
		if err != nil {
			return err
		}

		messageID := make([]byte, 20)
		if _, err := rand.Read(messageID); err != nil {
			return err
		}

		message := npb.Message{
			MessageType: npb.Message_ORDER,
			MessageID:   hex.EncodeToString(messageID),
			Payload:     payload,
		}

		vendor, err := peer.IDB58Decode(orderOpen.Listings[0].Listing.VendorID.PeerID)
		if err != nil {
			return err
		}

		if err := op.messenger.ReliablySendMessage(dbtx, vendor, &message, nil); err != nil {
			return err
		}

		if err := order.PutMessage(&payment); err != nil {
			return err
		}

		if funded {
			fundingTotal, err := order.FundingTotal()
			if err != nil {
				return err
			}
			notif := events.PaymentNotification{
				OrderID:      order.ID.String(),
				FundingTotal: fundingTotal.String(),
				CoinType:     orderOpen.Payment.Coin,
			}
			op.bus.Emit(&notif)
			log.Infof("Payment detected: Order %s fully funded", order.ID)
		} else {
			log.Infof("Payment detected: Order %s partially funded", order.ID)
		}

	case models.RoleVendor:
		if funded {
			notif := &events.OrderFundedNotification{
				BuyerHandle: orderOpen.BuyerID.Handle,
				BuyerID:     orderOpen.BuyerID.PeerID,
				ListingType: orderOpen.Listings[0].Listing.Metadata.ContractType.String(),
				OrderID:     order.ID.String(),
				Price: events.ListingPrice{
					Amount:        orderOpen.Payment.Amount,
					CurrencyCode:  orderOpen.Payment.Coin,
					PriceModifier: orderOpen.Listings[0].Listing.Item.CryptoListingPriceModifier,
				},
				Slug: orderOpen.Listings[0].Listing.Slug,
				Thumbnail: events.Thumbnail{
					Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
					Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
				},
				Title: orderOpen.Listings[0].Listing.Item.Title,
			}
			op.bus.Emit(&notif)
			log.Infof("Payment detected: Order %s fully funded", order.ID)
		} else {
			log.Infof("Payment detected: Order %s partially funded", order.ID)
		}
	}
	return nil
}

// processOutgoingPayment processes payments coming out of an order's payment address.
func (op *OrderProcessor) processOutgoingPayment(dbtx database.Tx, order *models.Order, tx iwallet.Transaction) error {
	err := order.PutTransaction(tx)
	if models.IsDuplicateTransactionError(err) {
		log.Debugf("Received duplicate transaction %s", tx.ID.String())
		return nil
	}
	return err
}

package orders

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
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

		if err := utils.SignOrderMessage(&resp, op.identityPrivateKey); err != nil {
			return err
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

		if err := order.PutMessage(&resp); err != nil {
			return err
		}

		if funded {
			fundingTotal, err := order.FundingTotal()
			if err != nil {
				return err
			}
			dbtx.RegisterCommitHook(func() {
				op.bus.Emit(&events.OrderPaymentReceived{
					OrderID:      order.ID.String(),
					FundingTotal: fundingTotal.String(),
					CoinType:     orderOpen.Payment.Coin,
				})
			})
			log.Infof("Payment detected: Order %s fully funded", order.ID)
		} else {
			log.Infof("Payment detected: Order %s partially funded", order.ID)
		}

	case models.RoleVendor:
		if funded {
			// TODO: mark vendor inventory downwards is not wasFunded.

			if err := op.sendRatingSignatures(dbtx, order, orderOpen); err != nil {
				log.Errorf("Error sending rating signature message: %s", err)
			}

			dbtx.RegisterCommitHook(func() {
				op.bus.Emit(&events.OrderFunded{
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
				})
			})
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
	dbtx.RegisterCommitHook(func() {
		op.bus.Emit(&events.SpendFromPaymentAddress{Transaction: tx})
	})
	return err
}

// checkForMorePayments loads open orders from the database and checks to see if it can find any more
// transactions relevant to the order. To do this it does the following:
// 1. Load all the order models that have transaction IDs
// 2. Query the wallet for each transaction ID without a transaction recorded for it.
// 3. Query the wallet for all transactions for the given address.
// 4. Process any new transactions found.
//
// Finally we check if the wallet implements the WalletScanner interface. If it does, we trigger a
// rescan for the order if one has never been performed before.
func (op *OrderProcessor) checkForMorePayments() {
	var (
		txs       []iwallet.Transaction
		rescanMap = make(map[iwallet.CoinType]time.Time)
	)
	err := op.db.Update(func(dbtx database.Tx) error {
		var orders []models.Order
		err := dbtx.Read().Where("open = ?", true).First(&orders).Error
		if err != nil && gorm.IsRecordNotFoundError(err) {
			return err
		}

		addressesToWatch := make(map[iwallet.CoinType][]iwallet.Address)

		for _, order := range orders {
			timestamp, err := order.Timestamp()
			if err != nil {
				log.Errorf("Error loading order timestamp %s", err)
				continue
			}

			orderOpen, err := order.OrderOpenMessage()
			if err != nil {
				log.Errorf("Error loading orderOpen message %s", err)
				continue
			}

			addrs, ok := addressesToWatch[iwallet.CoinType(orderOpen.Payment.Coin)]
			if ok {
				addrs = append(addrs, iwallet.NewAddress(orderOpen.Payment.Address, iwallet.CoinType(orderOpen.Payment.Coin)))
				addressesToWatch[iwallet.CoinType(orderOpen.Payment.Coin)] = addrs
			} else {
				addressesToWatch[iwallet.CoinType(orderOpen.Payment.Coin)] = []iwallet.Address{iwallet.NewAddress(orderOpen.Payment.Address, iwallet.CoinType(orderOpen.Payment.Coin))}
			}

			if !shouldWeQuery(timestamp, order.LastCheckForPayments) {
				continue
			}

			wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
			if err != nil {
				log.Errorf("Error loading wallet for order %s: %s", order.ID, err)
				continue
			}

			_, ok = wallet.(iwallet.WalletScanner)
			if ok && !order.RescanPerformed {
				earliest, ok := rescanMap[iwallet.CoinType(orderOpen.Payment.Coin)]
				if !ok || timestamp.Before(earliest) {
					rescanMap[iwallet.CoinType(orderOpen.Payment.Coin)] = timestamp
				}
				order.RescanPerformed = true
			}

			var missingTxids []iwallet.TransactionID

			knownTxs, err := order.GetTransactions()
			if err != nil && !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading known transactions: %s", err)
			}
			knownTxsMap := make(map[iwallet.TransactionID]bool)
			for _, tx := range knownTxs {
				knownTxsMap[tx.ID] = true
			}

			paymentMsgs, err := order.PaymentSentMessages()
			if err == nil {
				for _, msg := range paymentMsgs {
					txid := iwallet.TransactionID(msg.TransactionID)
					if !knownTxsMap[txid] {
						missingTxids = append(missingTxids, txid)
						knownTxsMap[txid] = true
					}
				}
			} else if !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading payment sent messages: %s", err)
			}

			refundMsgs, err := order.Refunds()
			if err == nil {
				for _, msg := range refundMsgs {
					if msg.GetTransactionID() != "" {
						txid := iwallet.TransactionID(msg.GetTransactionID())
						if !knownTxsMap[txid] {
							missingTxids = append(missingTxids, txid)
							knownTxsMap[txid] = true
						}
					}
				}
			} else if !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading refund messages: %s", err)
			}

			orderConfirmationMsg, err := order.OrderConfirmationMessage()
			if err == nil {
				if orderConfirmationMsg.TransactionID != "" {
					txid := iwallet.TransactionID(orderConfirmationMsg.TransactionID)
					if !knownTxsMap[txid] {
						missingTxids = append(missingTxids, txid)
						knownTxsMap[txid] = true
					}
				}
			} else if !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading order confirmation message: %s", err)
			}

			orderCancelMsg, err := order.OrderCancelMessage()
			if err == nil {
				if orderCancelMsg.TransactionID != "" {
					txid := iwallet.TransactionID(orderCancelMsg.TransactionID)
					if !knownTxsMap[txid] {
						missingTxids = append(missingTxids, txid)
						knownTxsMap[txid] = true
					}
				}
			} else if !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading order cancel message: %s", err)
			}

			disputeClosedMsg, err := order.DisputeClosedMessage()
			if err == nil {
				if disputeClosedMsg.TransactionID != "" {
					txid := iwallet.TransactionID(disputeClosedMsg.TransactionID)
					if !knownTxsMap[txid] {
						missingTxids = append(missingTxids, txid)
						knownTxsMap[txid] = true
					}
				}
			} else if !models.IsMessageNotExistError(err) {
				log.Errorf("Error loading dispute closed message: %s", err)
			}

			for _, missing := range missingTxids {
				tx, err := wallet.GetTransaction(missing)
				if err == nil {
					txs = append(txs, tx)
					knownTxsMap[missing] = true
				}
			}

			addrTxs, err := wallet.GetAddressTransactions(iwallet.NewAddress(orderOpen.Payment.Address, iwallet.CoinType(orderOpen.Payment.Coin)))
			if err == nil {
				for _, tx := range addrTxs {
					if !knownTxsMap[tx.ID] {
						txs = append(txs, tx)
					}
				}
			}
			order.LastCheckForPayments = time.Now()
			if err := dbtx.Save(&order); err != nil {
				log.Errorf("Error updating LastCheckForPayments: %s", err)
			}
		}

		for ct, addrs := range addressesToWatch {
			wallet, err := op.multiwallet.WalletForCurrencyCode(ct.CurrencyCode())
			if err != nil {
				return err
			}
			wtx, err := wallet.Begin()
			if err != nil {
				log.Errorf("Error saving watch address for coin %s: %s", ct.CurrencyCode(), err)
				continue
			}
			err = wallet.WatchAddress(wtx, addrs...)
			if err != nil {
				log.Errorf("Error saving watch address for coin %s: %s", ct.CurrencyCode(), err)
				continue
			}
			if err := wtx.Commit(); err != nil {
				log.Errorf("Error saving watch address for coin %s: %s", ct.CurrencyCode(), err)
				continue
			}
		}

		return nil
	})
	if err != nil {
		log.Errorf("Error checking for more payments: %s", err)
	}

	for _, tx := range txs {
		op.processWalletTransaction(tx)
	}

	for coin, timestamp := range rescanMap {
		wallet, err := op.multiwallet.WalletForCurrencyCode(coin.CurrencyCode())
		if err != nil {
			log.Errorf("Error loading wallet: %s", err)
			continue
		}
		scanner := wallet.(iwallet.WalletScanner)
		if err := scanner.RescanTransactions(timestamp.Add(-time.Hour*12), nil); err != nil {
			log.Errorf("Error starting rescan job: %s", err)
		}
	}
}

// shouldWeQuery calculates an exponential backoff for payment queries based
// on how old the order is and how long since our last attempt.
func shouldWeQuery(orderTimestamp time.Time, lastTry time.Time) bool {
	timeSinceMessage := time.Since(orderTimestamp)
	timeSinceLastTry := time.Since(lastTry)

	switch t := timeSinceMessage; {
	// Less than 1 week old order, retry every 10 minutes.
	case t < time.Minute*15 && timeSinceLastTry > time.Minute*10:
		return true
	// Less than 1 month old order, retry every hour.
	case t < time.Hour && timeSinceLastTry > time.Hour:
		return true
	// Less than 45 day old order, retry every 12 hours.
	case t < time.Hour*24 && timeSinceLastTry > time.Minute*12:
		return true
	// Less than six month old order, retry every 48 hours.
	case t < time.Hour*24*7 && timeSinceLastTry > time.Hour*48:
		return true
	// Less than one year old order, retry every week.
	case t < time.Hour*24*30 && timeSinceLastTry > time.Hour*24*7:
		return true
	// Older than 1 year old message, retry every 30 days.
	case t >= time.Hour*24*30*12 && timeSinceLastTry > time.Hour*24*30:
		return true
	}

	return false
}

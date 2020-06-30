package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-cid"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"testing"
	"time"
)

func TestOpenBazaarNode_PurchaseListing(t *testing.T) {
	// Create a new mock network with three nodes.
	network, err := NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	// Start the mock wallets for each node
	go network.StartWalletNetwork()

	// Start the order processors for each node.
	for _, node := range network.Nodes() {
		go node.orderProcessor.Start()
	}

	// Create message ack event subscription in node 1.
	ackSub1, err := network.Nodes()[1].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	// Create order event subscription for node 0.
	orderSub0, err := network.Nodes()[0].eventBus.Subscribe(&events.NewOrder{})
	if err != nil {
		t.Fatal(err)
	}

	// New mock listing from the factory.
	listing := factory.NewPhysicalListing("tshirt")

	// Save the listing in node 0 and block until saving is finished.
	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Fetch the listing index form node 0.
	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	// Set the profile on the moderator node (node 2) and block until saving is finished.
	done2 := make(chan struct{})
	if err := network.Nodes()[2].SetProfile(&models.Profile{Name: "Ron Paul"}, done2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done2:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Create and save the moderator info in node 2 and block until the saving is finished.
	modInfo := &models.ModeratorInfo{
		AcceptedCurrencies: []string{"MCK"},
		Fee: models.ModeratorFee{
			Percentage: 10,
			FeeType:    models.PercentageFee,
		},
	}
	done3 := make(chan struct{})
	if err := network.Nodes()[2].SetSelfAsModerator(context.Background(), modInfo, done3); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done3:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Create a purchase from the factory.
	purchase := factory.NewPurchase()
	purchase.Items[0].ListingHash = index[0].CID

	// Have node 1 purchase the listing from node 0.
	_, paymentAddress, paymentAmount, err := network.Nodes()[1].PurchaseListing(context.Background(), purchase)
	if err != nil {
		t.Fatal(err)
	}

	// Validate expected amount is correct.
	expectedAmount := "4992221"
	if paymentAmount.Amount.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount. Expected %s, got %s", expectedAmount, paymentAmount.Amount)
	}

	// Block until node 1 receives the message ACK for the purchase.
	select {
	case <-ackSub1.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
	// Block until node 0 receives the order message.
	var orderEvent interface{}
	select {
	case orderEvent = <-orderSub0.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Validate the event is correct.
	orderNotif := orderEvent.(*events.NewOrder)
	if orderNotif.BuyerID != network.Nodes()[1].Identity().Pretty() {
		t.Errorf("Incorrect notification peer ID: expected %s, got %s", network.Nodes()[1].Identity().Pretty(), orderNotif.BuyerID)
	}
	if orderNotif.Slug != listing.Slug {
		t.Errorf("Incorrect notification slug: expected %s, got %s", listing.Slug, orderNotif.Slug)
	}
	if orderNotif.Title != listing.Item.Title {
		t.Errorf("Incorrect notification title: expected %s, got %s", listing.Item.Title, orderNotif.Title)
	}
	if orderNotif.ListingType != listing.Metadata.ContractType.String() {
		t.Errorf("Incorrect notification listing type: expected %s, got %s", listing.Metadata.ContractType.String(), orderNotif.ListingType)
	}
	if orderNotif.Thumbnail.Small != listing.Item.Images[0].Small {
		t.Errorf("Incorrect notification small image: expected %s, got %s", listing.Item.Images[0].Small, orderNotif.Thumbnail.Small)
	}
	if orderNotif.Thumbnail.Tiny != listing.Item.Images[0].Tiny {
		t.Errorf("Incorrect notification tiny image: expected %s, got %s", listing.Item.Images[0].Tiny, orderNotif.Thumbnail.Tiny)
	}
	if orderNotif.Price.Amount == "" {
		t.Error("Order notification price not set")
	}
	if orderNotif.Price.CurrencyCode == "" {
		t.Error("Order notification currency code not set")
	}

	// Load the order from node 0 and validate it was indeed saved correctly.
	var order models.Order
	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderNotif.OrderID).Last(&order).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SerializedOrderOpen == nil {
		t.Error("Node 0 failed to save order")
	}

	// Load the order from node 1 and validate that it was indeed saved correctly.
	var order2 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderNotif.OrderID).Last(&order2).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order2.SerializedOrderOpen == nil {
		t.Error("Node 1 failed to save order")
	}

	// Validate that the order open ack was saved correctly.
	if !order2.OrderOpenAcked {
		t.Error("Node 1 failed to record order open ACK")
	}
	orderOpen, err := order2.OrderOpenMessage()
	if err != nil {
		t.Fatal(err)
	}

	// Validate the order is a direct payment.
	if orderOpen.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		t.Errorf("Expected direct order, got %s", orderOpen.Payment.Method)
	}

	// Generate some mock coins and send them to node 1.
	wallet, err := network.Nodes()[1].multiwallet.WalletForCurrencyCode("MCK")
	if err != nil {
		t.Fatal(err)
	}

	walletAddr, err := wallet.CurrentAddress()
	if err != nil {
		t.Fatal(err)
	}

	// Block until node 1 receives the mock coins.
	txSub, err := network.Nodes()[1].eventBus.Subscribe(&events.TransactionReceived{})
	if err != nil {
		t.Fatal(err)
	}

	if err := network.WalletNetwork().GenerateToAddress(walletAddr, iwallet.NewAmount(100000000)); err != nil {
		t.Fatal(err)
	}

	select {
	case <-txSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Send the payment for the order from node 1 to node 0 and block until node 1 detects the payment.
	paymentSub, err := network.Nodes()[1].eventBus.Subscribe(&events.OrderPaymentReceived{})
	if err != nil {
		t.Fatal(err)
	}

	dbtx, err := wallet.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = wallet.Spend(dbtx, paymentAddress, paymentAmount.Amount, iwallet.FlNormal)
	if err != nil {
		t.Fatal(err)
	}
	if err := dbtx.Commit(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-paymentSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Load the order from node 1 and make sure it is set to funded.
	var order5 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderNotif.OrderID).Last(&order5).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	funded, err := order5.IsFunded()
	if err != nil {
		t.Fatal(err)
	}
	if !funded {
		t.Errorf("Order not marked as funded in db")
	}

	// Next we're going to do the same but for a moderated order. Node 1 purchase a moderated
	// listing from node 0.
	purchase.Moderator = network.Nodes()[2].Identity().Pretty()
	_, _, paymentAmount, err = network.Nodes()[1].PurchaseListing(context.Background(), purchase)
	if err != nil {
		t.Fatal(err)
	}

	// Validate expected amount is correct.
	expectedAmount = "4992221"
	if paymentAmount.Amount.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount. Expected %s, got %s", expectedAmount, paymentAmount.Amount)
	}

	// Block until node 1 receives the order ACK.
	select {
	case <-ackSub1.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Block until node 0 receives the order.
	select {
	case orderEvent = <-orderSub0.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
	orderNotif = orderEvent.(*events.NewOrder)

	// Load the order from node 0 and make sure it was saved correctly.
	var order3 models.Order
	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderNotif.OrderID).Last(&order3).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order3.SerializedOrderOpen == nil {
		t.Error("Node 0 failed to save order")
	}

	// Load the order from node 1 and make sure it was saved correctly.
	var order4 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderNotif.OrderID).Last(&order4).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order4.SerializedOrderOpen == nil {
		t.Error("Node 1 failed to save order")
	}
	if !order4.OrderOpenAcked {
		t.Error("Node 1 failed to record order open ACK")
	}
	orderOpen, err = order4.OrderOpenMessage()
	if err != nil {
		t.Fatal(err)
	}

	// Validate that the order was saved as moderated.
	if orderOpen.Payment.Method != pb.OrderOpen_Payment_MODERATED {
		t.Errorf("Expected moderated order, got %s", orderOpen.Payment.Method)
	}

	// Finally we're going to make an offline direct order. Shut down node 0 so that it is offline.
	network.Nodes()[0].Stop(true)
	defer network.Nodes()[0].repo.DestroyRepo()
	network.nodes[0] = nil

	// Send the direct purchase from node 1 to node 0.
	purchase.Moderator = ""
	orderID, _, paymentAmount, err := network.Nodes()[1].PurchaseListing(context.Background(), purchase)
	if err != nil {
		t.Fatal(err)
	}

	// Validate the expected amount is correct.
	expectedAmount = "4992221"
	if paymentAmount.Amount.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount. Expected %s, got %s", expectedAmount, paymentAmount.Amount)
	}

	// Load the order from node 1 and make sure it was saved correctly.
	var order6 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order6).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order6.SerializedOrderOpen == nil {
		t.Error("Node 1 failed to save order")
	}
	orderOpen, err = order6.OrderOpenMessage()
	if err != nil {
		t.Fatal(err)
	}

	// Validate the order was saved as type cancelable.
	if orderOpen.Payment.Method != pb.OrderOpen_Payment_CANCELABLE {
		t.Errorf("Expected cancelable order, got %s", orderOpen.Payment.Method)
	}
}

func TestOpenBazaarNode_EstimateOrderSubtotal(t *testing.T) {
	network, err := NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	listing := factory.NewPhysicalListing("tshirt")

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	purchase := &models.Purchase{
		ShipTo:       "Peter",
		Address:      "123 Spooner St.",
		City:         "Quahog",
		State:        "RI",
		PostalCode:   "90210",
		CountryCode:  pb.CountryCode_UNITED_STATES.String(),
		AddressNotes: "asdf",
		Moderator:    "",
		Items: []models.PurchaseItem{
			{
				ListingHash: index[0].CID,
				Quantity:    "1",
				Options: []models.PurchaseItemOption{
					{
						Name:  "size",
						Value: "large",
					},
					{
						Name:  "color",
						Value: "red",
					},
				},
				Shipping: models.PurchaseShippingOption{
					Name:    "usps",
					Service: "standard",
				},
				Memo: "I want it fast!",
			},
		},
		AlternateContactInfo: "peter@protonmail.com",
		PaymentCoin:          "MCK",
	}

	val, err := network.Nodes()[1].EstimateOrderTotal(context.Background(), purchase)
	if err != nil {
		t.Fatal(err)
	}
	expectedAmount := 4992221
	if val.Total.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount: Expected %d, got %s", expectedAmount, val.Total)
	}
}

func TestOpenBazaarNode_createOrder(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer network.TearDown()

	listing := factory.NewPhysicalListing("tshirt")

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	lid, err := cid.Decode(index[0].CID)
	if err != nil {
		t.Fatal(err)
	}
	sl, err := network.Nodes()[1].GetListingByCID(context.Background(), lid)
	if err != nil {
		t.Fatal(err)
	}
	refundAddr := "abc"

	done2 := make(chan struct{})
	if err := network.Nodes()[1].SetProfile(&models.Profile{Name: "Ron Paul"}, done2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done2:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	modInfo := &models.ModeratorInfo{
		AcceptedCurrencies: []string{"MCK"},
		Fee: models.ModeratorFee{
			Percentage: 10,
			FeeType:    models.PercentageFee,
		},
	}
	done3 := make(chan struct{})
	if err := network.Nodes()[1].SetSelfAsModerator(context.Background(), modInfo, done3); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done3:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	tests := []struct {
		purchase   *models.Purchase
		checkOrder func(purchase *models.Purchase, order *pb.OrderOpen) error
	}{
		{
			// Successful physical good direct
			purchase: &models.Purchase{
				ShipTo:       "Peter",
				Address:      "123 Spooner St.",
				City:         "Quahog",
				State:        "RI",
				PostalCode:   "90210",
				CountryCode:  pb.CountryCode_UNITED_STATES.String(),
				AddressNotes: "asdf",
				Moderator:    "",
				Items: []models.PurchaseItem{
					{
						ListingHash: index[0].CID,
						Quantity:    "1",
						Options: []models.PurchaseItemOption{
							{
								Name:  "size",
								Value: "large",
							},
							{
								Name:  "color",
								Value: "red",
							},
						},
						Shipping: models.PurchaseShippingOption{
							Name:    "usps",
							Service: "standard",
						},
						Memo: "I want it fast!",
					},
				},
				AlternateContactInfo: "peter@protonmail.com",
				RefundAddress:        &refundAddr,
				PaymentCoin:          "MCK",
			},
			checkOrder: func(purchase *models.Purchase, order *pb.OrderOpen) error {
				if order.Shipping.ShipTo != purchase.ShipTo {
					return errors.New("incorrect ships to")
				}
				if order.Shipping.Address != purchase.Address {
					return errors.New("incorrect shipping address")
				}
				if order.Shipping.City != purchase.City {
					return errors.New("incorrect shipping city")
				}
				if order.Shipping.State != purchase.State {
					return errors.New("incorrect shipping state")
				}
				if order.Shipping.PostalCode != purchase.PostalCode {
					return errors.New("incorrect shipping postal code")
				}
				if order.Shipping.Country.String() != purchase.CountryCode {
					return errors.New("incorrect shipping country code")
				}
				if order.Shipping.AddressNotes != purchase.AddressNotes {
					return errors.New("incorrect shipping address notes")
				}
				if order.AlternateContactInfo != purchase.AlternateContactInfo {
					return errors.New("incorrect alternate contact info")
				}
				if order.Payment.Coin != purchase.PaymentCoin {
					return errors.New("incorrect payment coin")
				}
				if order.RefundAddress != *purchase.RefundAddress {
					return errors.New("incorrect refund address")
				}
				if len(order.Items) != 1 {
					return errors.New("incorrect number of items")
				}
				if len(order.Listings) != 1 {
					return errors.New("incorrect number of listings")
				}
				listingHash, err := utils.HashListing(sl)
				if err != nil {
					return err
				}
				orderListingHash, err := utils.HashListing(order.Listings[0])
				if err != nil {
					return err
				}
				if listingHash.B58String() != orderListingHash.B58String() {
					return errors.New("correct listing not included in order")
				}
				if order.Items[0].Quantity != purchase.Items[0].Quantity {
					return errors.New("incorrect quantity")
				}
				if order.Items[0].Memo != purchase.Items[0].Memo {
					return errors.New("incorrect memo")
				}
				if len(order.Items[0].Options) != 2 {
					return errors.New("incorrect number of options")
				}
				if order.Items[0].Options[0].Name != purchase.Items[0].Options[0].Name {
					return errors.New("incorrect option 0 name")
				}
				if order.Items[0].Options[1].Name != purchase.Items[0].Options[1].Name {
					return errors.New("incorrect option 1 name")
				}
				if order.Items[0].Options[0].Value != purchase.Items[0].Options[0].Value {
					return errors.New("incorrect value 0 name")
				}
				if order.Items[0].Options[1].Value != purchase.Items[0].Options[1].Value {
					return errors.New("incorrect value 1 name")
				}
				if order.Items[0].ShippingOption.Name != purchase.Items[0].Shipping.Name {
					return errors.New("incorrect shipping option name")
				}
				if order.Items[0].ShippingOption.Service != purchase.Items[0].Shipping.Service {
					return errors.New("incorrect shipping option service")
				}

				if order.BuyerID.PeerID != network.Nodes()[0].ipfsNode.Identity.Pretty() {
					return errors.New("incorrect buyer peer ID")
				}
				identityPubkey, err := crypto.MarshalPublicKey(network.Nodes()[0].ipfsNode.PrivateKey.GetPublic())
				if err != nil {
					return err
				}
				if !bytes.Equal(order.BuyerID.Pubkeys.Identity, identityPubkey) {
					return errors.New("incorrect buyer identity pubkey")
				}
				if !bytes.Equal(order.BuyerID.Pubkeys.Escrow, network.Nodes()[0].escrowMasterKey.PubKey().SerializeCompressed()) {
					return errors.New("incorrect buyer escrow pubkey")
				}

				sig, err := btcec.ParseSignature(order.BuyerID.Sig, btcec.S256())
				if err != nil {
					return err
				}
				idHash := sha256.Sum256([]byte(order.BuyerID.PeerID))
				valid := sig.Verify(idHash[:], network.Nodes()[0].escrowMasterKey.PubKey())
				if !valid {
					return errors.New("invalid buyer ID signature")
				}

				chaincode, err := hex.DecodeString(order.Payment.Chaincode)
				if err != nil {
					return err
				}
				keys, err := utils.GenerateRatingPublicKeys(network.Nodes()[0].ratingMasterKey.PubKey(), 1, chaincode)
				if err != nil {
					return err
				}
				if len(order.RatingKeys) != 1 {
					return errors.New("incorrect number of rating keys")
				}
				if !bytes.Equal(order.RatingKeys[0], keys[0]) {
					return errors.New("incorrect rating key in order")
				}

				if order.Payment.Amount != "4992221" {
					return errors.New("incorrect payment amount")
				}

				return nil
			},
		},
		{
			// Set refund address when nil
			purchase: &models.Purchase{
				Items: []models.PurchaseItem{
					{
						ListingHash: index[0].CID,
						Quantity:    "1",
						Options: []models.PurchaseItemOption{
							{
								Name:  "size",
								Value: "large",
							},
							{
								Name:  "color",
								Value: "red",
							},
						},
						Shipping: models.PurchaseShippingOption{
							Name:    "usps",
							Service: "standard",
						},
					},
				},
				PaymentCoin: "MCK",
			},
			checkOrder: func(purchase *models.Purchase, order *pb.OrderOpen) error {
				if order.RefundAddress == "" {
					return errors.New("refund address not set")
				}
				return nil
			},
		},
		{
			// Moderated order
			purchase: &models.Purchase{
				Items: []models.PurchaseItem{
					{
						ListingHash: index[0].CID,
						Quantity:    "1",
						Options: []models.PurchaseItemOption{
							{
								Name:  "size",
								Value: "large",
							},
							{
								Name:  "color",
								Value: "red",
							},
						},
						Shipping: models.PurchaseShippingOption{
							Name:    "usps",
							Service: "standard",
						},
					},
				},
				Moderator:   network.Nodes()[1].Identity().Pretty(),
				PaymentCoin: "MCK",
			},
			checkOrder: func(purchase *models.Purchase, order *pb.OrderOpen) error {
				if order.Payment.Moderator != network.Nodes()[1].ipfsNode.Identity.Pretty() {
					return errors.New("incorrect moderator set")
				}
				if order.Payment.Method != pb.OrderOpen_Payment_MODERATED {
					return errors.New("method not set as moderated")
				}
				if order.Payment.Script == "" {
					return errors.New("payment script not set")
				}

				var modKey []byte
				err := network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
					profile, err := tx.GetProfile()
					if err != nil {
						return err
					}
					modKey, err = hex.DecodeString(profile.EscrowPublicKey)
					return err
				})
				if err != nil {
					return err
				}

				if !bytes.Equal(order.Payment.ModeratorKey, modKey) {
					return errors.New("incorrect moderator key")
				}
				return nil
			},
		},
	}

	for i, test := range tests {
		order, err := network.Nodes()[0].createOrder(context.Background(), test.purchase)
		if err != nil {
			t.Errorf("Test %d: Failed to create order: %s", i, err)
			continue
		}
		if err := test.checkOrder(test.purchase, order); err != nil {
			t.Errorf("Test %d: Order check failed: %s", i, err)
		}
	}
}

func Test_createOrderUnkownVersion(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer network.TearDown()

	listing := factory.NewPhysicalListing("tshirt")
	listing.Metadata.Version = ListingVersion + 1

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	purchase := &models.Purchase{
		Items: []models.PurchaseItem{
			{
				ListingHash: index[0].CID,
				Quantity:    "1",
				Options: []models.PurchaseItemOption{
					{
						Name:  "size",
						Value: "large",
					},
					{
						Name:  "color",
						Value: "red",
					},
				},
				Shipping: models.PurchaseShippingOption{
					Name:    "usps",
					Service: "standard",
				},
			},
		},
		PaymentCoin: "MCK",
	}

	_, err = network.Nodes()[0].createOrder(context.Background(), purchase)
	if err != coreiface.ErrUnknownListingVersion {
		t.Errorf("Expected test to failed due to unknown listing version")
	}
}

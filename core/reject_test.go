package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"testing"
)

func TestOpenBazaarNode_RejectOrder(t *testing.T) {
	network, err := NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	go network.StartWalletNetwork()

	for _, node := range network.Nodes() {
		go node.orderProcessor.Start()
	}

	ackSub1, err := network.Nodes()[1].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	orderSub0, err := network.Nodes()[0].eventBus.Subscribe(&events.OrderNotification{})
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("tshirt")

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	done2 := make(chan struct{})
	if err := network.Nodes()[2].SetProfile(&models.Profile{Name: "Ron Paul"}, done2); err != nil {
		t.Fatal(err)
	}
	<-done2

	modInfo := &models.ModeratorInfo{
		AcceptedCurrencies: []string{"TMCK"},
		Fee: models.ModeratorFee{
			Percentage: 10,
			FeeType:    models.PercentageFee,
		},
	}
	done3 := make(chan struct{})
	if err := network.Nodes()[2].SetSelfAsModerator(context.Background(), modInfo, done3); err != nil {
		t.Fatal(err)
	}
	<-done3

	purchase := factory.NewPurchase()
	purchase.Items[0].ListingHash = index[0].Hash

	// Address request direct order
	_, paymentAddress, paymentAmount, err := network.Nodes()[1].PurchaseListing(purchase)
	if err != nil {
		t.Fatal(err)
	}

	expectedAmount := "4992221"
	if paymentAmount.Amount.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount. Expected %s, got %s", expectedAmount, paymentAmount.Amount)
	}

	<-ackSub1.Out()
	orderEvent := <-orderSub0.Out()
	orderNotif := orderEvent.(*events.OrderNotification)
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
	if !order2.OrderOpenAcked {
		t.Error("Node 1 failed to record order open ACK")
	}
	orderOpen, err := order2.OrderOpenMessage()
	if err != nil {
		t.Fatal(err)
	}
	if orderOpen.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		t.Errorf("Expected direct order, got %s", orderOpen.Payment.Method)
	}
}
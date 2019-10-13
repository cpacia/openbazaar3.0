package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"testing"
)

func TestOpenBazaarNode_ConfirmOrder(t *testing.T) {
	network, err := NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	go network.StartWalletNetwork()

	for _, node := range network.Nodes() {
		go node.orderProcessor.Start()
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
	orderID, _, _, err := network.Nodes()[1].PurchaseListing(purchase)
	if err != nil {
		t.Fatal(err)
	}

	<-orderSub0.Out()

	var order models.Order
	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SerializedOrderOpen == nil {
		t.Error("Node 0 failed to save order")
	}

	var order2 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order2).Error
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

	confirmSub, err := network.Nodes()[1].eventBus.Subscribe(&events.OrderConfirmationNotification{})
	if err != nil {
		t.Fatal(err)
	}

	confirmAck, err := network.Nodes()[0].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	done4 := make(chan struct{})
	if err := network.Nodes()[0].ConfirmOrder(orderID, done4); err != nil {
		t.Fatal(err)
	}
	<-done4

	<-confirmSub.Out()
	<-confirmAck.Out()

	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SerializedOrderConfirmation == nil {
		t.Error("Node 0 failed to save order confirmation")
	}
	if !order.OrderConfirmationAcked {
		t.Error("Node 0 failed to save order confirmation ack")
	}

	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order2).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order2.SerializedOrderConfirmation == nil {
		t.Error("Node 1 failed to save order confirmation")
	}
}

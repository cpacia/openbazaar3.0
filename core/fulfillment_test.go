package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	iwallet "github.com/cpacia/wallet-interface"
	"testing"
	"time"
)

func TestOpenBazaarNode_FufillOrder(t *testing.T) {
	network, err := NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	go network.StartWalletNetwork()

	for _, node := range network.Nodes() {
		go node.orderProcessor.Start()
	}

	done2 := make(chan struct{})
	if err := network.Nodes()[2].SetProfile(&models.Profile{Name: "Ron Paul"}, done2); err != nil {
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
	if err := network.Nodes()[2].SetSelfAsModerator(context.Background(), modInfo, done3); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done3:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	orderSub0, err := network.Nodes()[0].eventBus.Subscribe(&events.NewOrder{})
	if err != nil {
		t.Fatal(err)
	}
	orderAckSub0, err := network.Nodes()[1].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

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

	purchase := factory.NewPurchase()
	purchase.Items[0].ListingHash = index[0].Hash
	purchase.Moderator = network.Nodes()[2].Identity().Pretty()

	// Address request direct order
	orderID, paymentAddress, paymentAmount, err := network.Nodes()[1].PurchaseListing(context.Background(), purchase)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-orderSub0.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-orderAckSub0.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

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

	confirmSub, err := network.Nodes()[1].eventBus.Subscribe(&events.OrderConfirmation{})
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
	select {
	case <-done4:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-confirmSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
	select {
	case <-confirmAck.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

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

	wallet1, err := network.Nodes()[1].multiwallet.WalletForCurrencyCode(iwallet.CtMock)
	if err != nil {
		t.Fatal(err)
	}

	addr1, err := wallet1.CurrentAddress()
	if err != nil {
		t.Fatal(err)
	}

	txSub1, err := network.Nodes()[1].eventBus.Subscribe(&events.TransactionReceived{})
	if err != nil {
		t.Fatal(err)
	}

	if err := network.WalletNetwork().GenerateToAddress(addr1, iwallet.NewAmount(100000000000)); err != nil {
		t.Fatal(err)
	}

	select {
	case <-txSub1.Out():
		txSub1.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	fundingSub0, err := network.Nodes()[0].eventBus.Subscribe(&events.OrderFunded{})
	if err != nil {
		t.Fatal(err)
	}

	fundingSub1, err := network.Nodes()[1].eventBus.Subscribe(&events.OrderPaymentReceived{})
	if err != nil {
		t.Fatal(err)
	}

	wTx, err := wallet1.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wallet1.Spend(wTx, paymentAddress, paymentAmount.Amount, iwallet.FlNormal); err != nil {
		t.Fatal(err)
	}

	if err := wTx.Commit(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-fundingSub0.Out():
		fundingSub0.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-fundingSub1.Out():
		fundingSub1.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	fulfillSub, err := network.Nodes()[1].eventBus.Subscribe(&events.OrderFulfillment{})
	if err != nil {
		t.Fatal(err)
	}

	fulfillAck, err := network.Nodes()[0].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	done5 := make(chan struct{})
	fulfillments := []models.Fulfillment{
		{
			ItemIndex: 0,
			PhysicalDelivery: &models.PhysicalDelivery{
				TrackingNumber: "1234",
				Shipper:        "UPS",
			},
		},
	}
	if err := network.Nodes()[0].FulfillOrder(orderID, fulfillments, done5); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done5:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-fulfillSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
	select {
	case <-fulfillAck.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SerializedOrderFulfillments == nil {
		t.Error("Node 0 failed to save order fulfillment")
	}
	if !order.OrderFulfillmentAcked {
		t.Error("Node 0 failed to save order fulfillment ack")
	}

	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Last(&order2).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order2.SerializedOrderFulfillments == nil {
		t.Error("Node 1 failed to save order fulfillment")
	}

	fulfillmentMessages, err := order.OrderFulfillmentMessages()
	if err != nil {
		t.Fatal(err)
	}

	if len(fulfillmentMessages) != 1 {
		t.Errorf("Expected 1 saved fulfillment message, got %d", len(fulfillmentMessages))
	}

	if len(fulfillmentMessages[0].Fulfillments) != 1 {
		t.Errorf("Expected 1 saved fulfillment message, got %d", len(fulfillmentMessages[0].Fulfillments))
	}

	if fulfillmentMessages[0].Fulfillments[0].ItemIndex != 0 {
		t.Errorf("Expected item index of 0 got %d", fulfillmentMessages[0].Fulfillments[0].ItemIndex)
	}

	if fulfillmentMessages[0].Fulfillments[0].GetPhysicalDelivery().Shipper != "UPS" {
		t.Errorf("Expected shipper of UPS got %s", fulfillmentMessages[0].Fulfillments[0].GetPhysicalDelivery().Shipper)
	}

	if fulfillmentMessages[0].Fulfillments[0].GetPhysicalDelivery().TrackingNumber != "1234" {
		t.Errorf("Expected tracking number of 1234 got %s", fulfillmentMessages[0].Fulfillments[0].GetPhysicalDelivery().TrackingNumber)
	}

	if len(fulfillmentMessages[0].ReleaseInfo.FromIDs) != 1 {
		t.Errorf("Expected 1 from ID got %d", len(fulfillmentMessages[0].ReleaseInfo.FromIDs))
	}

	if len(fulfillmentMessages[0].ReleaseInfo.EscrowSignatures) != 1 {
		t.Errorf("Expected 1 signature got %d", len(fulfillmentMessages[0].ReleaseInfo.EscrowSignatures))
	}
}

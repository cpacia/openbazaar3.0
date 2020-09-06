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

func TestOpenBazaarNode_OpenDispute(t *testing.T) {
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

	done1 := make(chan struct{})
	if err := network.Nodes()[1].SetProfile(&models.Profile{Name: "Buyer"}, done1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done1:
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
	purchase.Items[0].ListingHash = index[0].CID
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

	ratingSigAck, err := network.Nodes()[0].eventBus.Subscribe(&events.MessageACK{})
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

	select {
	case <-ratingSigAck.Out():
		ratingSigAck.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	caseOpenSub, err := network.Nodes()[2].eventBus.Subscribe(&events.CaseOpen{})
	if err != nil {
		t.Fatal(err)
	}
	caseUpdateSub, err := network.Nodes()[2].eventBus.Subscribe(&events.CaseUpdate{})
	if err != nil {
		t.Fatal(err)
	}

	caseUpdateAckSub, err := network.Nodes()[0].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	disputeOpenSub, err := network.Nodes()[0].eventBus.Subscribe(&events.DisputeOpen{})
	if err != nil {
		t.Fatal(err)
	}

	disputeOpenAckModeratorSub, err := network.Nodes()[1].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	disputeOpenAckVendorSub, err := network.Nodes()[1].eventBus.Subscribe(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	done5 := make(chan struct{})
	if err := network.Nodes()[1].OpenDispute(orderID, "Got scammed", done5); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done5:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-disputeOpenSub.Out():
		disputeOpenSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-caseOpenSub.Out():
		caseOpenSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-caseUpdateSub.Out():
		caseUpdateSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-disputeOpenAckModeratorSub.Out():
		disputeOpenAckModeratorSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-disputeOpenAckVendorSub.Out():
		disputeOpenAckVendorSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-caseUpdateAckSub.Out():
		caseUpdateAckSub.Close()
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	var order1 models.Order
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id =?", orderID.String()).First(&order1).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order1.SerializedDisputeOpen == nil {
		t.Error("Buyer dispute open is nil")
	}
	if order1.DisputeOpenOtherPartyAcked == false {
		t.Error("Buyer dispute open other party ack is false")
	}
	if order1.DisputeOpenModeratorAcked == false {
		t.Error("Buyer dispute open moderator ack is false")
	}

	var order0 models.Order
	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id =?", orderID.String()).First(&order0).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if order0.SerializedDisputeOpen == nil {
		t.Error("Vendor dispute open is nil")
	}
	if order0.SerializedDisputeUpdate == nil {
		t.Error("Vendor dispute update is nil")
	}
	if order0.DisputeUpdateAcked == false {
		t.Error("Vendor dispute update ack is false")
	}

	var case2 models.Case
	err = network.Nodes()[2].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).First(&case2).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if case2.SerializedDisputeOpen == nil {
		t.Error("Moderator dispute open is nil")
	}
	if case2.BuyerContract == nil {
		t.Error("Moderator buyer contract is nil")
	}
	if case2.VendorContract == nil {
		t.Error("Moderator vendor contract is nil")
	}
}

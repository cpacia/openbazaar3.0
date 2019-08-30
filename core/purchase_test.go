package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"testing"
)

func TestOpenBazaarNode_PurchaseListing(t *testing.T) {
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
	<-done

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
				ListingHash: index[0].Hash,
				Quantity:    1,
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
		PaymentCoin:          "TMCK",
	}

	_, _, paymentAmount, err := network.Nodes()[1].PurchaseListing(purchase)
	if err != nil {
		t.Fatal(err)
	}

	expectedAmount := "4992221"
	if paymentAmount.Amount.Cmp(iwallet.NewAmount(expectedAmount)) != 0 {
		t.Errorf("Returned incorrect amount. Expected %s, got %s", expectedAmount, paymentAmount.Amount)
	}

	// TODO: check order saved correctly on both sides
	// TODO: check buyer saved order ACK

	// TODO: try again with cancelable order
	// TODO: try again with moderated order
}

func TestOpenBazaarNode_EstimateOrderSubtotal(t *testing.T) {

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
	<-done

	index, err := network.Nodes()[0].GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	sl, err := network.Nodes()[0].GetMyListingBySlug("tshirt")
	if err != nil {
		t.Fatal(err)
	}
	refundAddr := "abc"

	done2 := make(chan struct{})
	if err := network.Nodes()[1].SetProfile(&models.Profile{Name: "Ron Paul"}, done2); err != nil {
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
	if err := network.Nodes()[1].SetSelfAsModerator(context.Background(), modInfo, done3); err != nil {
		t.Fatal(err)
	}
	<-done3

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
						ListingHash: index[0].Hash,
						Quantity:    1,
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
				PaymentCoin:          "TMCK",
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
						ListingHash: index[0].Hash,
						Quantity:    1,
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
				PaymentCoin: "TMCK",
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
						ListingHash: index[0].Hash,
						Quantity:    1,
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
				PaymentCoin: "TMCK",
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
		order, err := network.Nodes()[0].createOrder(test.purchase)
		if err != nil {
			t.Errorf("Test %d: Failed to create order: %s", i, err)
			continue
		}
		if err := test.checkOrder(test.purchase, order); err != nil {
			t.Errorf("Test %d: Order check failed: %s", i, err)
		}
	}
}

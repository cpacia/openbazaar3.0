package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"testing"
)

func TestOpenBazaarNode_PurchaseListing(t *testing.T) {

}

func TestOpenBazaarNode_EstimateOrderSubtotal(t *testing.T) {

}

func TestOpenBazaarNode_createOrder(t *testing.T) {
	mockNode, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("tshirt")

	done := make(chan struct{})
	if err := mockNode.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	index, err := mockNode.GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	sl, err := mockNode.GetMyListingBySlug("tshirt")
	if err != nil {
		t.Fatal(err)
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
				RefundAddress:        nil,
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
				if order.RefundAddress == "" {
					return errors.New("refund address not set")
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

				if order.BuyerID.PeerID != mockNode.ipfsNode.Identity.Pretty() {
					return errors.New("incorrect buyer peer ID")
				}
				identityPubkey, err := crypto.MarshalPublicKey(mockNode.ipfsNode.PrivateKey.GetPublic())
				if err != nil {
					return err
				}
				if !bytes.Equal(order.BuyerID.Pubkeys.Identity, identityPubkey) {
					return errors.New("incorrect buyer identity pubkey")
				}
				if !bytes.Equal(order.BuyerID.Pubkeys.Escrow, mockNode.escrowMasterKey.PubKey().SerializeCompressed()) {
					return errors.New("incorrect buyer escrow pubkey")
				}

				chaincode, err := hex.DecodeString(order.Payment.Chaincode)
				if err != nil {
					return err
				}
				keys, err := utils.GenerateRatingPublicKeys(mockNode.ratingMasterKey.PubKey(), 1, chaincode)
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
	}

	for i, test := range tests {
		order, err := mockNode.createOrder(test.purchase)
		if err != nil {
			t.Errorf("Test %d: Failed to create order: %s", i, err)
		}
		if err := test.checkOrder(test.purchase, order); err != nil {
			t.Errorf("Test %d: Order check failed: %s", i, err)
		}
	}
}

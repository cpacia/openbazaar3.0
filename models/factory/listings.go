package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
)

func NewPhysicalListing(slug string) *pb.Listing {
	return &pb.Listing{
		Slug:               slug,
		TermsAndConditions: "Sample Terms and Conditions",
		RefundPolicy:       "Sample Refund policy",
		Metadata: &pb.Listing_Metadata{
			Version:            1,
			AcceptedCurrencies: []string{"MCK"},
			PricingCurrency: &pb.Currency{
				Code:         "USD",
				Divisibility: 2,
			},
			Expiry:             &timestamp.Timestamp{Seconds: 2147483647},
			Format:             pb.Listing_Metadata_FIXED_PRICE,
			ContractType:       pb.Listing_Metadata_PHYSICAL_GOOD,
			EscrowTimeoutHours: 1080,
		},
		Item: &pb.Listing_Item{
			Title: "Ron Swanson Tshirt",
			Tags:  []string{"tshirts"},
			Options: []*pb.Listing_Item_Option{
				{
					Name:        "Size",
					Description: "What size do you want your shirt?",
					Variants: []*pb.Listing_Item_Option_Variant{
						{Name: "Small", Image: NewImage()},
						{Name: "Large", Image: NewImage()},
					},
				},
				{
					Name:        "Color",
					Description: "What color do you want your shirt?",
					Variants: []*pb.Listing_Item_Option_Variant{
						{Name: "Red", Image: NewImage()},
						{Name: "Green", Image: NewImage()},
					},
				},
			},
			Nsfw:           false,
			Description:    "Example item",
			Price:          "100",
			ProcessingTime: "3 days",
			Categories:     []string{"tshirts"},
			Grams:          14,
			Condition:      "new",
			Images:         []*pb.Listing_Item_Image{NewImage(), NewImage()},
			Skus: []*pb.Listing_Item_Sku{
				{
					Selections: []*pb.Listing_Item_Sku_Selection{
						{
							Option:  "Size",
							Variant: "Large",
						},
						{
							Option:  "Color",
							Variant: "Red",
						},
					},
					Surcharge: "0",
					Quantity:  "12",
					ProductID: "1",
				},
				{
					Surcharge: "0",
					Quantity:  "44",
					ProductID: "2",
					Selections: []*pb.Listing_Item_Sku_Selection{
						{
							Option:  "Size",
							Variant: "Small",
						},
						{
							Option:  "Color",
							Variant: "Green",
						},
					},
				},
			},
		},
		Taxes: []*pb.Listing_Tax{
			{
				Percentage:  7,
				TaxShipping: true,
				TaxType:     "Sales tax",
				TaxRegions:  []pb.CountryCode{pb.CountryCode_UNITED_STATES},
			},
		},
		ShippingOptions: []*pb.Listing_ShippingOption{
			{
				Name:    "usps",
				Type:    pb.Listing_ShippingOption_FIXED_PRICE,
				Regions: []pb.CountryCode{pb.CountryCode_ALL},
				Services: []*pb.Listing_ShippingOption_Service{
					{
						Name:              "standard",
						Price:             "20",
						EstimatedDelivery: "3 days",
					},
				},
			},
		},
		Coupons: []*pb.Listing_Coupon{
			{
				Title:    "Insider's Discount",
				Code:     &pb.Listing_Coupon_DiscountCode{DiscountCode: "insider"},
				Discount: &pb.Listing_Coupon_PercentDiscount{PercentDiscount: 5},
			},
		},
	}
}

func NewDigitalListing(slug string) *pb.Listing {
	return &pb.Listing{
		Slug:               slug,
		TermsAndConditions: "Sample Terms and Conditions",
		RefundPolicy:       "Sample Refund policy",
		Metadata: &pb.Listing_Metadata{
			Version:            1,
			AcceptedCurrencies: []string{"MCK"},
			PricingCurrency: &pb.Currency{
				Code:         "USD",
				Divisibility: 2,
			},
			Expiry:       &timestamp.Timestamp{Seconds: 2147483647},
			Format:       pb.Listing_Metadata_FIXED_PRICE,
			ContractType: pb.Listing_Metadata_DIGITAL_GOOD,
		},
		Item: &pb.Listing_Item{
			Title:          "Ron Swanson image",
			Tags:           []string{"pics"},
			Nsfw:           false,
			Description:    "Example item",
			Price:          "100",
			ProcessingTime: "3 days",
			Categories:     []string{"pics"},
			Grams:          14,
			Condition:      "new",
			Images:         []*pb.Listing_Item_Image{NewImage(), NewImage()},
			Skus: []*pb.Listing_Item_Sku{
				{
					Surcharge: "0",
					Quantity:  "12",
					ProductID: "1",
				},
			},
		},
		Taxes: []*pb.Listing_Tax{
			{
				Percentage:  7,
				TaxShipping: true,
				TaxType:     "Sales tax",
				TaxRegions:  []pb.CountryCode{pb.CountryCode_UNITED_STATES},
			},
		},
		Coupons: []*pb.Listing_Coupon{
			{
				Title:    "Insider's Discount",
				Code:     &pb.Listing_Coupon_DiscountCode{DiscountCode: "insider"},
				Discount: &pb.Listing_Coupon_PercentDiscount{PercentDiscount: 5},
			},
		},
	}
}

func NewCryptoListing(slug string) *pb.Listing {
	listing := NewPhysicalListing(slug)
	listing.Item.CryptoListingCurrencyCode = "TETH"
	listing.Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY
	listing.Item.Skus = []*pb.Listing_Item_Sku{{Quantity: "100000000"}}
	listing.ShippingOptions = nil
	listing.Item.Condition = ""
	listing.Item.Options = nil
	listing.Item.Price = "100"
	listing.Coupons = nil
	return listing
}

func NewSignedListing() *pb.SignedListing {
	privKeyBytes, _ := hex.DecodeString("08011240522f4983cfc829245dff096e41fc6f56a84a4ec960db89af6d746e98d38bf9bd0667074541e46bfc5fdcc68d0a80a26fc50875178a3af4c2f30c7199dbf452c7")
	privkey, _ := crypto.UnmarshalPrivateKey(privKeyBytes)
	pubkeyBytes, _ := privkey.GetPublic().Bytes()
	pid, _ := peer.IDFromPublicKey(privkey.GetPublic())

	escrowPrivkeyBytes, _ := hex.DecodeString("781405f8b9f4000d3f3dc319c2ed82be6c5812c0f8d7bd086a5bfe1930f3225e")
	escrowPrivkey, escrowPubkey := btcec.PrivKeyFromBytes(btcec.S256(), escrowPrivkeyBytes)

	sigHash := sha256.Sum256([]byte(pid.Pretty()))
	sig, _ := escrowPrivkey.Sign(sigHash[:])

	listing := NewPhysicalListing("ron-swanson-shirt")
	listing.VendorID = &pb.ID{
		PeerID: pid.Pretty(),
		Pubkeys: &pb.ID_Pubkeys{
			Identity: pubkeyBytes,
			Escrow:   escrowPubkey.SerializeCompressed(),
		},
		Sig: sig.Serialize(),
	}

	ser, _ := proto.Marshal(listing)
	listingSig, _ := privkey.Sign(ser)

	return &pb.SignedListing{Listing: listing, Signature: listingSig}
}

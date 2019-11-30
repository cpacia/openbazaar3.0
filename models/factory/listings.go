package factory

import (
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes/timestamp"
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
			Expiry:       &timestamp.Timestamp{Seconds: 2147483647},
			Format:       pb.Listing_Metadata_FIXED_PRICE,
			ContractType: pb.Listing_Metadata_PHYSICAL_GOOD,
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
	j := `{
    "listing": {
        "slug": "ron-swanson-shirt",
        "vendorID": {
            "peerID": "12D3KooWFKi1TgtdCSn571Sv6jkud58dneiSeVmUjxRM4QgJEAfc",
            "pubkeys": {
                "identity": "CAESIFHOthBNLNHwxcQC+mYhZtB9xO1Xd8iIMQToUm0elMu7",
                "escrow": "Agpdqnu/m8Qq8cJuJJGCnkT0ZosVuVJ0Lqy76oDQsqmY"
            },
            "sig": "MEQCICklJuGy2/FlXAcY5fkFtrBQR1F1BIUewsY7/0BOgRokAiAdDxeGk8/i9Av2n23Yq0OwAIfnhL8iAYFpzRbef1iRfg=="
        },
        "metadata": {
            "version": 1,
            "contractType": "PHYSICAL_GOOD",
            "format": "FIXED_PRICE",
            "expiry": "2038-01-19T03:14:07.000Z",
            "acceptedCurrencies": [
                "MCK"
            ],
            "pricingCurrency": {
                "code": "USD",
                "divisibility": 2
            }
        },
        "item": {
            "title": "Ron Swanson Tshirt",
            "description": "Example item",
            "processingTime": "3 days",
            "tags": [
                "tshirts"
            ],
            "images": [
                {
                    "filename": "image.jpg",
                    "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                },
                {
                    "filename": "image.jpg",
                    "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                    "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                }
            ],
            "categories": [
                "tshirts"
            ],
            "grams": 14,
            "condition": "new",
            "options": [
                {
                    "name": "Size",
                    "description": "What size do you want your shirt?",
                    "variants": [
                        {
                            "name": "Small",
                            "image": {
                                "filename": "image.jpg",
                                "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                            }
                        },
                        {
                            "name": "Large",
                            "image": {
                                "filename": "image.jpg",
                                "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                            }
                        }
                    ]
                },
                {
                    "name": "Color",
                    "description": "What color do you want your shirt?",
                    "variants": [
                        {
                            "name": "Red",
                            "image": {
                                "filename": "image.jpg",
                                "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                            }
                        },
                        {
                            "name": "Green",
                            "image": {
                                "filename": "image.jpg",
                                "original": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "large": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "medium": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "small": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
                                "tiny": "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub"
                            }
                        }
                    ]
                }
            ],
            "skus": [
                {
                    "selections": [
                        {
                            "option": "Size",
                            "variant": "Large"
                        },
                        {
                            "option": "Color",
                            "variant": "Red"
                        }
                    ],
                    "productID": "1",
                    "quantity": "12",
                    "surcharge": "0"
                },
                {
                    "selections": [
                        {
                            "option": "Size",
                            "variant": "Small"
                        },
                        {
                            "option": "Color",
                            "variant": "Green"
                        }
                    ],
                    "productID": "2",
                    "quantity": "44",
                    "surcharge": "0"
                }
            ],
            "price": "100"
        },
        "shippingOptions": [
            {
                "name": "usps",
                "type": "FIXED_PRICE",
                "regions": [
                    "ALL"
                ],
                "services": [
                    {
                        "name": "standard",
                        "estimatedDelivery": "3 days",
                        "price": "20"
                    }
                ]
            }
        ],
        "taxes": [
            {
                "taxType": "Sales tax",
                "taxRegions": [
                    "UNITED_STATES"
                ],
                "taxShipping": true,
                "percentage": 7
            }
        ],
        "coupons": [
            {
                "title": "Insider's Discount",
                "discountCode": "insider",
                "percentDiscount": 5
            }
        ],
        "termsAndConditions": "Sample Terms and Conditions",
        "refundPolicy": "Sample Refund policy"
    },
    "signature": "ef+mD7WagP43eXBH6J/CwjmNYvBRgz1sopb19ZV7LkKi5xTvEcadbA1U6QwWVPK/Is+RQbtuRDgpXjmbuqLeBA=="
}`
	sl := new(pb.SignedListing)
	jsonpb.UnmarshalString(j, sl)

	return sl
}

package orders

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"testing"
)

func Test_convertCurrencyAmount(t *testing.T) {
	erp, err := wallet.NewMockExchangeRates()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		amount           string
		originalCurrency string
		paymentCurrency  string
		expected         string
	}{
		{ // Exchange rate $407
			"100",
			"USD",
			"BCH",
			"245579",
		},
		{ // Same currency
			"100000",
			"BCH",
			"BCH",
			"100000",
		},
		{ // Exchange rate 31.588915
			"100000000",
			"BTC",
			"BCH",
			"3158891949",
		},
		{
			"500000000",
			"LTC",
			"BCH",
			"140816694",
		},
		{
			"100",
			"USD",
			"MCK",
			"3888024",
		},
	}

	for i, test := range tests {
		original, err := models.CurrencyDefinitions.Lookup(test.originalCurrency)
		if err != nil {
			t.Fatal(err)
		}

		payment, err := models.CurrencyDefinitions.Lookup(test.paymentCurrency)
		if err != nil {
			t.Fatal(err)
		}

		amount, err := convertCurrencyAmount(models.NewCurrencyValue(test.amount, original), payment, erp)
		if err != nil {
			t.Errorf("Test %d failed: %s", i, err)
			continue
		}

		if amount.String() != test.expected {
			t.Errorf("Test %d returned incorrect amount. Expected %s, got %s", i, test.expected, amount.String())
		}
	}
}

func TestCalculateOrderTotal(t *testing.T) {
	tests := []struct {
		transform     func(order *pb.OrderOpen) error
		expectedTotal iwallet.Amount
	}{
		{ // Normal
			transform:     func(order *pb.OrderOpen) error { return nil },
			expectedTotal: iwallet.NewAmount("4992221"),
		},
		{ // Quantity 2
			transform: func(order *pb.OrderOpen) error {
				order.Items[0].Quantity = 2
				return nil
			},
			expectedTotal: iwallet.NewAmount("9152406"),
		},
		{ // Additional item shipping
			transform: func(order *pb.OrderOpen) error {
				order.Listings[0].Listing.ShippingOptions[0].Services[0].AdditionalItemPrice = "20"
				hash, err := hashListing(order.Listings[0])
				if err != nil {
					return err
				}
				order.Items[0].Quantity = 2
				order.Items[0].ListingHash = hash.B58String()
				return nil
			},
			expectedTotal: iwallet.NewAmount("9984442"),
		},
		{ // Multiple items
			transform: func(order *pb.OrderOpen) error {
				order.Listings = append(order.Listings, order.Listings[0])
				order.Listings[1].Listing.Item.Title = "abc"
				order.Listings[1].Listing.ShippingOptions[0].Services[0].Price = "30"
				hash, err := hashListing(order.Listings[1])
				if err != nil {
					return err
				}
				order.Items = append(order.Items, order.Items[0])
				order.Items[1].ListingHash = hash.B58String()
				return nil
			},
			expectedTotal: iwallet.NewAmount("9568425"),
		},
		{ // Coupons
			transform: func(order *pb.OrderOpen) error {
				order.Items[0].CouponCodes = []string{
					"insider",
				}
				return nil
			},
			expectedTotal: iwallet.NewAmount("4784212"),
		},
		{ // Price Discount
			transform: func(order *pb.OrderOpen) error {
				order.Listings = append(order.Listings, order.Listings[0])
				order.Listings[1].Listing.Item.Title = "abc"
				order.Listings[1].Listing.Coupons[0].Discount = &pb.Listing_Coupon_PriceDiscount{PriceDiscount: "5"}
				hash, err := hashListing(order.Listings[1])
				if err != nil {
					return err
				}
				order.Items[0].ListingHash = hash.B58String()
				order.Items[0].CouponCodes = []string{
					"insider",
				}
				return nil
			},
			expectedTotal: iwallet.NewAmount("4784212"),
		},
		{ // Market price listing
			transform: func(order *pb.OrderOpen) error {
				order.Listings[0].Listing.Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY
				order.Listings[0].Listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				order.Listings[0].Listing.Metadata.PricingCurrency = &pb.Currency{
					Code:         "BTC",
					Divisibility: 8,
					Name:         "Bitcoin Cash",
					CurrencyType: "Crypto",
				}
				order.Listings[0].Listing.ShippingOptions = nil
				order.Listings[0].Listing.Taxes = nil
				hash, err := hashListing(order.Listings[0])
				if err != nil {
					return err
				}
				order.Items[0].ListingHash = hash.B58String()
				order.Items[0].Quantity = 10000
				order.Items[0].ShippingOption = nil
				return nil
			},
			expectedTotal: iwallet.NewAmount("5000025"),
		},
	}

	erp, err := wallet.NewMockExchangeRates()
	if err != nil {
		t.Fatal(err)
	}
	for i, test := range tests {
		order, _, err := factory.NewOrder()
		if err != nil {
			t.Fatal(err)
		}
		if err := test.transform(order); err != nil {
			t.Errorf("Error transforming listing in test %d: %s", i, err)
			continue
		}
		total, err := CalculateOrderTotal(order, erp)
		if err != nil {
			t.Errorf("Error calculating total for test %d: %s", i, err)
			continue
		}
		if total.Cmp(test.expectedTotal) != 0 {
			t.Errorf("Incorrect order total for test %d. Expected %s, got %s", i, test.expectedTotal, total)
		}
	}
}

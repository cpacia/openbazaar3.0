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
		transform     func(order *pb.OrderOpen)
		expectedTotal iwallet.Amount
	}{
		{
			transform:     func(order *pb.OrderOpen) {},
			expectedTotal: iwallet.NewAmount("4938333"),
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
		test.transform(order)
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

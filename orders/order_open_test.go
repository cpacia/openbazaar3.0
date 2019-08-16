package orders

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/wallet"
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

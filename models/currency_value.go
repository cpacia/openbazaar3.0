package models

import (
	"math/big"
)

type CurrencyValue struct {
	Amount       *big.Int `json:"amount"`
	CurrencyCode string   `json:"currencyCode"`
}

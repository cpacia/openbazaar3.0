package core

import (
	"errors"
	"fmt"
)

var (
	// ErrListingCoinDivisibilityIncorrect - coin divisibility err
	ErrListingCoinDivisibilityIncorrect = errors.New("incorrect coinDivisibility")
)

type ErrTooManyItems []string

func (e ErrTooManyItems) Error() string {
	return fmt.Sprintf("field: %s has a size greater than the max of %s", e[0], e[1])
}

type ErrTooManyCharacters []string

func (e ErrTooManyCharacters) Error() string {
	return fmt.Sprintf("field: %s has a length greater than the max of %s", e[0], e[1])
}

type ErrMissingField string

func (e ErrMissingField) Error() string {
	return fmt.Sprintf("missing field: %s", string(e))
}

// ErrPriceModifierOutOfRange - customize limits for price modifier
type ErrPriceModifierOutOfRange struct {
	Min float64
	Max float64
}

func (e ErrPriceModifierOutOfRange) Error() string {
	return fmt.Sprintf("priceModifier out of range: [%.2f, %.2f]", e.Min, e.Max)
}

// ErrCryptocurrencyListingIllegalField - invalid field err
type ErrCryptocurrencyListingIllegalField string

func (e ErrCryptocurrencyListingIllegalField) Error() string {
	return illegalFieldString("cryptocurrency listing", string(e))
}

// ErrMarketPriceListingIllegalField - invalid listing field err
type ErrMarketPriceListingIllegalField string

func (e ErrMarketPriceListingIllegalField) Error() string {
	return illegalFieldString("market price listing", string(e))
}

func illegalFieldString(objectType string, field string) string {
	return fmt.Sprintf("Illegal %s field: %s", objectType, field)
}

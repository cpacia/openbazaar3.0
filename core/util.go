package core

import (
	"github.com/cpacia/openbazaar3.0/models"
)

// NormalizeCurrencyCode standardizes the format for the given currency code.
func normalizeCurrencyCode(currencyCode string) string {
	var c, err = models.CurrencyDefinitions.Lookup(currencyCode)
	if err != nil {
		log.Errorf("invalid currency code (%s): %s", currencyCode, err.Error())
		return ""
	}
	return c.String()
}

// maybeCloseDone is a helper to close the done chan if it's not nil.
func maybeCloseDone(done chan<- struct{}) {
	if done != nil {
		close(done)
	}
}

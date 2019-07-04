package core

import (
	"crypto/sha256"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/multiformats/go-multihash"
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

// multihashSha256 hashes the data with sha256 and returns a multihash representation.
func multihashSha256(b []byte) (*multihash.Multihash, error) {
	h := sha256.Sum256(b)
	encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return nil, err
	}
	multihash, err := multihash.Cast(encoded)
	if err != nil {
		return nil, err
	}
	return &multihash, err
}

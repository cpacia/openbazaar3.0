package utils

import (
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"testing"
)

func TestMultihashSha256(t *testing.T) {
	h := "c560021782de34f597ef1b2bd415d20c7febe7f111e6c1da349990323e082c74"
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	mh, err := MultihashSha256(b)
	if err != nil {
		t.Fatal(err)
	}

	expected := "QmRPibjjnNE4FrvbKzADHr98PEopP6Mv1Zzw48C2atEt6q"
	if mh.B58String() != expected {
		t.Errorf("Incorrect hash returned expected %s, got %s", expected, mh.B58String())
	}
}

func TestHashListing(t *testing.T) {
	sl := factory.NewSignedListing()

	mh, err := HashListing(sl)
	if err != nil {
		t.Fatal(err)
	}

	expected := "QmVQHxZLBqaZgocUxw47GUW3HC1BHiV9WUfDGKPZPmmJ32"
	if mh.B58String() != expected {
		t.Errorf("Incorrect hash returned expected %s, got %s", expected, mh.B58String())
	}
}

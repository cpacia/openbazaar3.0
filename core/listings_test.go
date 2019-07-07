package core

import (
	"github.com/cpacia/openbazaar3.0/models/factory"
	"testing"
)

func TestOpenBazaarNode_SaveListing(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := node.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
}

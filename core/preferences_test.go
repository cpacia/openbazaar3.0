package core

import (
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"testing"
	"time"
)

func TestOpenBazaarNode_SavePreferences(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}
	defer node.repo.DestroyRepo()


	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := node.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	prefs := models.UserPreferences{
		RefundPolicy: "asdf",
	}

	if err := node.SavePreferences(&prefs, nil); err != nil {
		t.Fatal(err)
	}

	var savedPrefs models.UserPreferences
	err = node.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().First(&savedPrefs).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if savedPrefs.RefundPolicy != prefs.RefundPolicy {
		t.Errorf("Expected refund policy %s, got %s", prefs.RefundPolicy, savedPrefs.RefundPolicy)
	}

	prefs = models.UserPreferences{
		Blocked: []byte(`["aasdf"]`),
	}

	if err := node.SavePreferences(&prefs, nil); err == nil {
		t.Errorf("Expected error got nil")
	}

	prefs = models.UserPreferences{
		Mods: []byte(`["aasdf"]`),
	}

	if err := node.SavePreferences(&prefs, nil); err == nil {
		t.Errorf("Expected error got nil")
	}

	mods := []string{"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"}
	out, err := json.Marshal(mods)
	if err != nil {
		t.Fatal(err)
	}
	prefs = models.UserPreferences{
		Mods: out,
	}

	if err := node.SavePreferences(&prefs, nil); err != nil {
		t.Fatal(err)
	}

	sl, err := node.GetMyListingBySlug("ron-swanson-shirt")
	if err != nil {
		t.Fatal(err)
	}
	if len(sl.Listing.GetModerators()) != 1 {
		t.Errorf("Expected 1 mod got %d", len(sl.Listing.GetModerators()))
	}
	if sl.Listing.GetModerators()[0] != mods[0] {
		t.Errorf("Expected moderator %s, got %s", mods[0], sl.Listing.GetModerators()[0])
	}
}

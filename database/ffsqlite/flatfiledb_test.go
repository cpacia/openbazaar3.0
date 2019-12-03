package ffsqlite

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"os"
	"path"
	"testing"
)

func TestNewFlatFIleDB(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "public_test")
	_, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	directories := []string{
		path.Join(dir, "listings"),
		path.Join(dir, "ratings"),
		path.Join(dir, "images"),
		path.Join(dir, "images", "tiny"),
		path.Join(dir, "images", "small"),
		path.Join(dir, "images", "medium"),
		path.Join(dir, "images", "large"),
		path.Join(dir, "images", "original"),
		path.Join(dir, "posts"),
		path.Join(dir, "files"),
	}

	for _, p := range directories {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("Failed to create directory %s", p)
		}
	}
}

func TestFlatFileDB_Profile(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "profile_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := "Ron Swanson"
	err = fdb.SetProfile(&models.Profile{
		Name: name,
	})
	if err != nil {
		t.Fatal(err)
	}

	pro, err := fdb.GetProfile()
	if err != nil {
		t.Fatal(err)
	}

	if pro.Name != name {
		t.Errorf("Incorrect name returned. Expected %s got %s", name, pro.Name)
	}
}

func TestFlatFileDB_Followers(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "followers_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	l := models.Followers{
		"QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
		"Qmd9hFFuueFrSR7YwUuAfirXXJ7ANZAMc5sx4HFxn7mPkc",
	}
	err = fdb.SetFollowers(l)
	if err != nil {
		t.Fatal(err)
	}

	followers, err := fdb.GetFollowers()
	if err != nil {
		t.Fatal(err)
	}

	if followers[0] != l[0] {
		t.Errorf("Incorrect peerID returned. Expected %s got %s", l[0], followers[0])
	}
	if followers[1] != l[1] {
		t.Errorf("Incorrect peerID returned. Expected %s got %s", l[1], followers[1])
	}
}

func TestFlatFileDB_Following(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "following_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	l := models.Following{
		"Qmd9hFFuueFrSR7YwUuAfirXXJ7ANZAMc5sx4HFxn7mPkc",
		"QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
	}
	err = fdb.SetFollowing(l)
	if err != nil {
		t.Fatal(err)
	}

	following, err := fdb.GetFollowing()
	if err != nil {
		t.Fatal(err)
	}

	if following[0] != l[0] {
		t.Errorf("Incorrect peerID returned. Expected %s got %s", l[0], following[0])
	}
	if following[1] != l[1] {
		t.Errorf("Incorrect peerID returned. Expected %s got %s", l[1], following[1])
	}
}

func TestFlatFileDB_Listing(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "listing_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		slug   = "test-listing"
		policy = "test-policy"
	)
	sl := &pb.SignedListing{
		Listing: &pb.Listing{
			Slug:         slug,
			RefundPolicy: policy,
		},
	}
	err = fdb.SetListing(sl)
	if err != nil {
		t.Fatal(err)
	}

	listing, err := fdb.GetListing(slug)
	if err != nil {
		t.Fatal(err)
	}

	if listing.Listing.Slug != slug {
		t.Errorf("Incorrect listing returned. Expected slug %s got %s", slug, listing.Listing.Slug)
	}

	if listing.Listing.RefundPolicy != policy {
		t.Errorf("Incorrect listing returned. Expected policy %s got %s", policy, listing.Listing.RefundPolicy)
	}

	sl, err = fdb.getSignedListing(slug)
	if err != nil {
		t.Fatal(err)
	}

	if sl.Listing.Slug != slug {
		t.Errorf("Incorrect listing returned. Expected slug %s got %s", slug, listing.Listing.Slug)
	}

	if sl.Listing.RefundPolicy != policy {
		t.Errorf("Incorrect listing returned. Expected policy %s got %s", policy, listing.Listing.RefundPolicy)
	}

	if err := fdb.DeleteListing(slug); err != nil {
		t.Fatal(err)
	}
	_, err = fdb.GetListing(slug)
	if !os.IsNotExist(err) {
		t.Errorf("Expected os not exist error, got %s", err)
	}
}

func TestFlatFileDB_ListingIndex(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "listingIndex_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		slug1 = "test-listing1"
		slug2 = "test-listing2"
		h1    = "111"
		h2    = "222"
	)
	i := models.ListingIndex{
		{
			Slug: slug1,
			Hash: h1,
		},
		{
			Slug: slug2,
			Hash: h2,
		},
	}
	err = fdb.SetListingIndex(i)
	if err != nil {
		t.Fatal(err)
	}

	index, err := fdb.GetListingIndex()
	if err != nil {
		t.Fatal(err)
	}

	if index[0].Slug != slug1 {
		t.Errorf("Incorrect index returned. Expected slug %s got %s", slug1, index[0].Slug)
	}
	if index[1].Slug != slug2 {
		t.Errorf("Incorrect index returned. Expected slug %s got %s", slug2, index[1].Slug)
	}
	if index[0].Hash != h1 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h1, index[0].Hash)
	}
	if index[1].Hash != h2 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h2, index[1].Hash)
	}
}

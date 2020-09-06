package ffsqlite

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
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
			CID:  h1,
		},
		{
			Slug: slug2,
			CID:  h2,
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
	if index[0].CID != h1 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h1, index[0].CID)
	}
	if index[1].CID != h2 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h2, index[1].CID)
	}
}

func TestFlatFileDB_RatingIndex(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "ratingIndex_test")
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
	i := models.RatingIndex{
		{
			Slug: slug1,
			Ratings: []string{
				h1,
			},
		},
		{
			Slug: slug2,
			Ratings: []string{
				h2,
			},
		},
	}
	err = fdb.SetRatingIndex(i)
	if err != nil {
		t.Fatal(err)
	}

	index, err := fdb.GetRatingIndex()
	if err != nil {
		t.Fatal(err)
	}

	if index[0].Slug != slug1 {
		t.Errorf("Incorrect index returned. Expected slug %s got %s", slug1, index[0].Slug)
	}
	if index[1].Slug != slug2 {
		t.Errorf("Incorrect index returned. Expected slug %s got %s", slug2, index[1].Slug)
	}
	if index[0].Ratings[0] != h1 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h1, index[0].Ratings[0])
	}
	if index[1].Ratings[0] != h2 {
		t.Errorf("Incorrect index returned. Expected hash %s got %s", h2, index[1].Ratings[0])
	}
}

func TestFlatFileDB_Rating(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "rating_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r := &pb.Rating{
		Overall: 5,
	}
	err = fdb.SetRating(r)
	if err != nil {
		t.Fatal(err)
	}

	ser, err := proto.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	h, err := utils.MultihashSha256(ser)
	if err != nil {
		t.Fatal(err)
	}

	f, err := ioutil.ReadFile(path.Join(fdb.rootDir, "ratings", h.B58String()[:16]+".json"))
	if err != nil {
		t.Fatal(err)
	}

	r2 := new(pb.Rating)
	err = jsonpb.UnmarshalString(string(f), r2)
	if err != nil {
		t.Fatal(err)
	}

	if r2.Overall != 5 {
		t.Errorf("Expected overall of 5 got %d", r2.Overall)
	}
}

func TestFlatFileDB_Images(t *testing.T) {
	dir := path.Join(os.TempDir(), "openbazaar", "rating_test")
	fdb, err := NewFlatFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := fdb.SetImage([]byte{0x00}, path.Join(fdb.rootDir, "images", string(models.ImageSizeOriginal), "test")); err != nil {
		t.Fatal(err)
	}

	// Make sure duplicate succeeds.
	if err := fdb.SetImage([]byte{0x00}, path.Join(fdb.rootDir, "images", string(models.ImageSizeOriginal), "test")); err != nil {
		t.Fatal(err)
	}

	// Check exists
	_, err = os.Stat(path.Join(fdb.rootDir, "images", string(models.ImageSizeOriginal), "test"))
	if os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

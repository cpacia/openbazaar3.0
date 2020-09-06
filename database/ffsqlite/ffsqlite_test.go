package ffsqlite

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestFFSqliteDB_UpdateAndView(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-update")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx database.Tx) error {
		if err := tx.Migrate(&models.OutgoingMessage{}); err != nil {
			return err
		}
		return tx.Save(&models.OutgoingMessage{ID: "abc"})
	})
	if err != nil {
		t.Error(err)
	}

	var messages []models.OutgoingMessage
	err = db.View(func(tx database.Tx) error {
		if err := tx.Read().Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 1 {
		t.Errorf("Db update failed. Expected %d messages got %d", 1, len(messages))
	}

	err = db.Update(func(tx database.Tx) error {
		err := errors.New("atomic update failure")

		if err := tx.Save(&models.OutgoingMessage{ID: "abc"}); err != nil {
			t.Fatal(err)
		}
		return err
	})
	if err == nil {
		t.Error("Update function did not return error")
	}

	var messages2 []models.OutgoingMessage
	err = db.View(func(tx database.Tx) error {
		if err := tx.Read().Find(&messages2).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) > 1 {
		t.Error("Db update failed to roll back.")
	}
}

func TestFFSqliteDB_Rollback(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-update")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx database.Tx) error {
		return tx.Migrate(&models.OutgoingMessage{})
	})
	if err != nil {
		t.Fatal(err)
	}

	name := "Ron Paul"
	err = db.Update(func(tx database.Tx) error {
		if err := tx.Save(&models.OutgoingMessage{ID: "abc"}); err != nil {
			return err
		}
		if err := tx.SetProfile(&models.Profile{Name: name}); err != nil {
			return err
		}
		return errors.New("failure :(")
	})
	if err == nil {
		t.Error("no error returned from update")
	}

	var (
		messages []models.OutgoingMessage
		profile  *models.Profile
	)
	err = db.View(func(tx database.Tx) error {
		if err := tx.Read().Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		profile, err = tx.GetProfile()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 0 {
		t.Error("Db update failed to roll back.")
	}

	if profile != nil {
		t.Error("Db update failed to roll back.")
	}
}

func TestFFSqliteDB_CRUD(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-update")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx database.Tx) error {
		if err := tx.Migrate(&models.ChatMessage{}); err != nil {
			return err
		}
		return tx.Save(&models.ChatMessage{
			MessageID: "abc",
			PeerID:    "qm123",
			OrderID:   "test",
			Timestamp: time.Time{},
			Read:      false,
			Outgoing:  false,
			Message:   "hello",
			Sequence:  0,
		})
	})
	if err != nil {
		t.Error(err)
	}

	var messages []models.ChatMessage
	err = db.View(func(tx database.Tx) error {
		return tx.Read().Find(&messages).Error
	})
	if err != nil {
		t.Error(err)
	}

	if len(messages) != 1 {
		t.Error("Failed to save message to the database")
	}

	err = db.Update(func(tx database.Tx) error {
		return tx.Update("read", true, map[string]interface{}{"peer_id = ?": "qm123", "order_id = ?": "test"}, &models.ChatMessage{})
	})
	if err != nil {
		t.Error(err)
	}

	var messages2 []models.ChatMessage
	err = db.View(func(tx database.Tx) error {
		return tx.Read().Find(&messages2).Error
	})
	if err != nil {
		t.Error(err)
	}

	if len(messages2) != 1 {
		t.Error("Failed to read message to the database")
	}

	if !messages2[0].Read {
		t.Error("Failed to update model to set read to true")
	}

	err = db.Update(func(tx database.Tx) error {
		return tx.Delete("peer_id", "qm123", nil, &models.ChatMessage{})
	})
	if err != nil {
		t.Error(err)
	}

	var messages3 []models.ChatMessage
	err = db.View(func(tx database.Tx) error {
		return tx.Read().Find(&messages3).Error
	})
	if err != nil {
		t.Error(err)
	}

	if len(messages3) != 0 {
		t.Error("Failed to delete chat message from the database")
	}
}

func TestFFSqliteDB_profile(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-profile")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		name  = "Ron Paul"
		name2 = "Ron Paul2"
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetProfile(&models.Profile{Name: name}); err != nil {
			return err
		}
		if err := tx.SetProfile(&models.Profile{Name: name2}); err != nil {
			return err
		}
		profile, err := tx.GetProfile()
		if err != nil {
			return err
		}
		if profile.Name != name2 {
			t.Errorf("Returned incorrect profile name. Expected %s, got %s", name2, profile.Name)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var profile *models.Profile
	err = db.View(func(tx database.Tx) error {
		profile, err = tx.GetProfile()
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if profile.Name != name2 {
		t.Errorf("Returned incorrect profile name. Expected %s, got %s", name2, profile.Name)
	}
}

func TestFFSqliteDB_followers(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-followers")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		follower1 = "f1"
		follower2 = "f2"
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetFollowers(models.Followers{follower1}); err != nil {
			return err
		}
		if err := tx.SetFollowers(models.Followers{follower1, follower2}); err != nil {
			return err
		}
		followers, err := tx.GetFollowers()
		if err != nil {
			return err
		}
		if len(followers) != 2 {
			t.Errorf("Expected 2 followers, got %d", len(followers))
		}
		if followers[0] != follower1 {
			t.Errorf("Returned incorrect followers. Expected %s, got %s", follower1, followers[0])
		}
		if followers[1] != follower2 {
			t.Errorf("Returned incorrect followers. Expected %s, got %s", follower2, followers[1])
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var followers models.Followers
	err = db.View(func(tx database.Tx) error {
		followers, err = tx.GetFollowers()
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if len(followers) != 2 {
		t.Errorf("Expected 2 followers, got %d", len(followers))
	}
	if followers[0] != follower1 {
		t.Errorf("Returned incorrect followers. Expected %s, got %s", follower1, followers[0])
	}
	if followers[1] != follower2 {
		t.Errorf("Returned incorrect followers. Expected %s, got %s", follower2, followers[1])
	}
}

func TestFFSqliteDB_following(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-following")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		following1 = "f1"
		following2 = "f2"
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetFollowing(models.Following{following1}); err != nil {
			return err
		}
		if err := tx.SetFollowing(models.Following{following1, following2}); err != nil {
			return err
		}
		following, err := tx.GetFollowing()
		if err != nil {
			return err
		}
		if len(following) != 2 {
			t.Errorf("Expected 2 followers, got %d", len(following))
		}
		if following[0] != following1 {
			t.Errorf("Returned incorrect followers. Expected %s, got %s", following1, following[0])
		}
		if following[1] != following2 {
			t.Errorf("Returned incorrect followers. Expected %s, got %s", following2, following[1])
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var following models.Following
	err = db.View(func(tx database.Tx) error {
		following, err = tx.GetFollowing()
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if len(following) != 2 {
		t.Errorf("Expected 2 followers, got %d", len(following))
	}
	if following[0] != following1 {
		t.Errorf("Returned incorrect followers. Expected %s, got %s", following1, following[0])
	}
	if following[1] != following2 {
		t.Errorf("Returned incorrect followers. Expected %s, got %s", following2, following[1])
	}
}

func TestFFSqliteDB_listing(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-listing")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		listing1 = &pb.SignedListing{
			Listing: &pb.Listing{
				Slug:               "slug1",
				TermsAndConditions: "terms1",
			},
		}
		listing2 = &pb.SignedListing{
			Listing: &pb.Listing{
				Slug:               "slug1",
				TermsAndConditions: "terms2",
			},
		}
		listing3 = &pb.SignedListing{
			Listing: &pb.Listing{
				Slug:               "slug2",
				TermsAndConditions: "terms2",
			},
		}
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetListing(listing1); err != nil {
			return err
		}
		if err := tx.SetListing(listing2); err != nil {
			return err
		}
		if err := tx.SetListing(listing3); err != nil {
			return err
		}
		l1, err := tx.GetListing(listing1.Listing.Slug)
		if err != nil {
			return err
		}
		if l1.Listing.Slug != listing1.Listing.Slug {
			t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing1.Listing.Slug, l1.Listing.Slug)
		}
		if l1.Listing.TermsAndConditions != listing2.Listing.TermsAndConditions {
			t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing2.Listing.TermsAndConditions, l1.Listing.TermsAndConditions)
		}
		l3, err := tx.GetListing(listing3.Listing.Slug)
		if err != nil {
			return err
		}
		if l3.Listing.Slug != listing3.Listing.Slug {
			t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing3.Listing.Slug, l3.Listing.Slug)
		}
		if l3.Listing.TermsAndConditions != listing3.Listing.TermsAndConditions {
			t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing3.Listing.TermsAndConditions, l3.Listing.TermsAndConditions)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var (
		l1 *pb.SignedListing
		l3 *pb.SignedListing
	)
	err = db.View(func(tx database.Tx) error {
		l1, err = tx.GetListing(listing1.Listing.Slug)
		if err != nil {
			return err
		}
		l3, err = tx.GetListing(listing3.Listing.Slug)
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if l1.Listing.Slug != listing1.Listing.Slug {
		t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing1.Listing.Slug, l1.Listing.Slug)
	}
	if l1.Listing.TermsAndConditions != listing2.Listing.TermsAndConditions {
		t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing2.Listing.TermsAndConditions, l1.Listing.TermsAndConditions)
	}
	if l3.Listing.Slug != listing3.Listing.Slug {
		t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing3.Listing.Slug, l3.Listing.Slug)
	}
	if l3.Listing.TermsAndConditions != listing3.Listing.TermsAndConditions {
		t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing3.Listing.TermsAndConditions, l3.Listing.TermsAndConditions)
	}

	err = db.Update(func(tx database.Tx) error {
		return tx.DeleteListing(l1.Listing.Slug)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.View(func(tx database.Tx) error {
		l1, err = tx.GetListing(l1.Listing.Slug)
		if !os.IsNotExist(err) {
			t.Error("Deleted listing still exists")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFFSqliteDB_listingIndex(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-listingIndex")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		index1 = models.ListingIndex{
			{
				Slug: "slug1",
			},
			{
				Slug: "slug2",
			},
		}
		index2 = models.ListingIndex{
			{
				Slug: "slug3",
			},
			{
				Slug: "slug4",
			},
		}
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetListingIndex(index1); err != nil {
			return err
		}
		if err := tx.SetListingIndex(index2); err != nil {
			return err
		}

		index, err := tx.GetListingIndex()
		if err != nil {
			return err
		}
		if index[0].Slug != index2[0].Slug {
			t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[0].Slug, index[0].Slug)
		}
		if index[1].Slug != index2[1].Slug {
			t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[1].Slug, index[1].Slug)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var index models.ListingIndex
	err = db.View(func(tx database.Tx) error {
		index, err = tx.GetListingIndex()
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if index[0].Slug != index2[0].Slug {
		t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[0].Slug, index[0].Slug)
	}
	if index[1].Slug != index2[1].Slug {
		t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[1].Slug, index[1].Slug)
	}
}

func TestFFSqliteDB_rating(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-rating")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	//defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		rating1 = &pb.Rating{
			VendorSig: &pb.RatingSignature{
				Slug: "slug0",
			},
			Overall: 5,
		}
		rating2 = &pb.Rating{
			VendorSig: &pb.RatingSignature{
				Slug: "slug1",
			},
			Overall: 4,
		}
		rating3 = &pb.Rating{
			VendorSig: &pb.RatingSignature{
				Slug: "slug2",
			},
			Overall: 3,
		}
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetRating(rating1); err != nil {
			return err
		}
		if err := tx.SetRating(rating2); err != nil {
			return err
		}
		if err := tx.SetRating(rating3); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	ser, err := proto.Marshal(rating1)
	if err != nil {
		t.Fatal(err)
	}
	h, err := utils.MultihashSha256(ser)
	if err != nil {
		t.Fatal(err)
	}

	f, err := ioutil.ReadFile(path.Join(dataDir, "public", "ratings", h.B58String()[:16]+".json"))
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

	if r2.VendorSig.Slug != rating1.VendorSig.Slug {
		t.Errorf("Expected slug of %s got %s", rating1.VendorSig.Slug, r2.VendorSig.Slug)
	}

	ser, err = proto.Marshal(rating2)
	if err != nil {
		t.Fatal(err)
	}
	h, err = utils.MultihashSha256(ser)
	if err != nil {
		t.Fatal(err)
	}

	f, err = ioutil.ReadFile(path.Join(dataDir, "public", "ratings", h.B58String()[:16]+".json"))
	if err != nil {
		t.Fatal(err)
	}

	r2 = new(pb.Rating)
	err = jsonpb.UnmarshalString(string(f), r2)
	if err != nil {
		t.Fatal(err)
	}

	if r2.Overall != 4 {
		t.Errorf("Expected overall of 4 got %d", r2.Overall)
	}

	if r2.VendorSig.Slug != rating2.VendorSig.Slug {
		t.Errorf("Expected slug of %s got %s", rating2.VendorSig.Slug, r2.VendorSig.Slug)
	}

	ser, err = proto.Marshal(rating3)
	if err != nil {
		t.Fatal(err)
	}
	h, err = utils.MultihashSha256(ser)
	if err != nil {
		t.Fatal(err)
	}

	f, err = ioutil.ReadFile(path.Join(dataDir, "public", "ratings", h.B58String()[:16]+".json"))
	if err != nil {
		t.Fatal(err)
	}

	r2 = new(pb.Rating)
	err = jsonpb.UnmarshalString(string(f), r2)
	if err != nil {
		t.Fatal(err)
	}

	if r2.Overall != 3 {
		t.Errorf("Expected overall of 3 got %d", r2.Overall)
	}

	if r2.VendorSig.Slug != rating3.VendorSig.Slug {
		t.Errorf("Expected slug of %s got %s", rating3.VendorSig.Slug, r2.VendorSig.Slug)
	}
}

func TestFFSqliteDB_ratingIndex(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-ratingIndex")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		index1 = models.RatingIndex{
			{
				Slug: "slug1",
			},
			{
				Slug: "slug2",
			},
		}
		index2 = models.RatingIndex{
			{
				Slug: "slug3",
			},
			{
				Slug: "slug4",
			},
		}
	)
	err = db.Update(func(tx database.Tx) error {
		if err := tx.SetRatingIndex(index1); err != nil {
			return err
		}
		if err := tx.SetRatingIndex(index2); err != nil {
			return err
		}

		index, err := tx.GetRatingIndex()
		if err != nil {
			return err
		}
		if index[0].Slug != index2[0].Slug {
			t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[0].Slug, index[0].Slug)
		}
		if index[1].Slug != index2[1].Slug {
			t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[1].Slug, index[1].Slug)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var index models.RatingIndex
	err = db.View(func(tx database.Tx) error {
		index, err = tx.GetRatingIndex()
		return err
	})
	if err != nil {
		t.Error(err)
	}
	if index[0].Slug != index2[0].Slug {
		t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[0].Slug, index[0].Slug)
	}
	if index[1].Slug != index2[1].Slug {
		t.Errorf("Returned incorred index. Expected slug %s, got %s", index2[1].Slug, index[1].Slug)
	}
}

func TestFFSqliteDB_Images(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-images")

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	db, err := NewFFMemoryDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx database.Tx) error {
		err := tx.SetImage(models.Image{
			ImageBytes: []byte{0x00},
			Size:       models.ImageSizeOriginal,
			Name:       "image1",
		})
		if err != nil {
			return err
		}
		err = tx.SetImage(models.Image{
			ImageBytes: []byte{0x01},
			Size:       models.ImageSizeOriginal,
			Name:       "image2",
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(path.Join(db.PublicDataPath(), "images", string(models.ImageSizeOriginal), "image1"))
	if os.IsNotExist(err) {
		t.Error("File was not created")
	}

	_, err = os.Stat(path.Join(db.PublicDataPath(), "images", string(models.ImageSizeOriginal), "image2"))
	if os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

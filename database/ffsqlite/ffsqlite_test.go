package ffsqlite

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/jinzhu/gorm"
	"os"
	"path"
	"testing"
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
		if err := tx.DB().AutoMigrate(&models.OutgoingMessage{}).Error; err != nil {
			return err
		}
		return tx.DB().Save(&models.OutgoingMessage{ID: "abc"}).Error
	})
	if err != nil {
		t.Error(err)
	}

	var messages []models.OutgoingMessage
	err = db.View(func(tx database.Tx) error {
		if err := tx.DB().Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
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

		if err := tx.DB().Save(&models.OutgoingMessage{ID: "abc"}).Error; err != nil {
			t.Fatal(err)
		}
		return err
	})
	if err == nil {
		t.Error("Update function did not return error")
	}

	var messages2 []models.OutgoingMessage
	err = db.View(func(tx database.Tx) error {
		if err := tx.DB().Find(&messages2).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
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
		return tx.DB().AutoMigrate(&models.OutgoingMessage{}).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	name := "Ron Paul"
	err = db.Update(func(tx database.Tx) error {
		if err := tx.DB().Save(&models.OutgoingMessage{ID: "abc"}).Error; err != nil {
			return err
		}
		if err := tx.SetProfile(&models.Profile{Name:name}); err != nil {
			return err
		}
		return errors.New("failure :(")
	})
	if err == nil {
		t.Error("no error returned from update")
	}

	var (
		messages []models.OutgoingMessage
		profile *models.Profile
	)
	err = db.View(func(tx database.Tx) error {
		if err := tx.DB().Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
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

	if profile.Name == name {
		t.Error("Db update failed to roll back.")
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
		if l1.Slug != listing1.Listing.Slug {
			t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing1.Listing.Slug, l1.Slug)
		}
		if l1.TermsAndConditions != listing2.Listing.TermsAndConditions {
			t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing2.Listing.TermsAndConditions, l1.TermsAndConditions)
		}
		l3, err := tx.GetListing(listing3.Listing.Slug)
		if err != nil {
			return err
		}
		if l3.Slug != listing3.Listing.Slug {
			t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing3.Listing.Slug, l3.Slug)
		}
		if l3.TermsAndConditions != listing3.Listing.TermsAndConditions {
			t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing3.Listing.TermsAndConditions, l3.TermsAndConditions)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	var (
		l1 *pb.Listing
		l3 *pb.Listing
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
	if l1.Slug != listing1.Listing.Slug {
		t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing1.Listing.Slug, l1.Slug)
	}
	if l1.TermsAndConditions != listing2.Listing.TermsAndConditions {
		t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing2.Listing.TermsAndConditions, l1.TermsAndConditions)
	}
	if l3.Slug != listing3.Listing.Slug {
		t.Errorf("Returned incorrect listing slug. Expected %s, got %s", listing3.Listing.Slug, l3.Slug)
	}
	if l3.TermsAndConditions != listing3.Listing.TermsAndConditions {
		t.Errorf("Returned incorrect listing terms. Expected %s, got %s", listing3.Listing.TermsAndConditions, l3.TermsAndConditions)
	}

	err = db.Update(func(tx database.Tx)error {
		return tx.DeleteListing(l1.Slug)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.View(func(tx database.Tx) error {
		l1, err = tx.GetListing(l1.Slug)
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

package ffsqlite

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/jinzhu/gorm"
	"os"
	"path"
	"testing"
)

func TestFFSqliteDB_UpdateAndView(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "ffsqlitedb-update")

	if err := os.Mkdir(dataDir, os.ModePerm); err != nil {
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
	err = db.View(func(tx database.Tx)error {
		if err := tx.DB().Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 1 {
		t.Error("Db update failed to roll back.")
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
	err = db.View(func(tx database.Tx)error {
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

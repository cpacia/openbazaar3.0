package repo

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/jinzhu/gorm"
	"sync"
	"testing"
)

func TestSqliteDB_Update(t *testing.T) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sdb := &SqliteDB{db, sync.RWMutex{}}

	if err := autoMigrateDatabase(db); err != nil {
		t.Fatal(err)
	}

	err = sdb.Update(func(tx *gorm.DB)error {
		return tx.Save(&models.OutgoingMessage{ID:"abc"}).Error
	})
	if err != nil {
		t.Error(err)
	}

	var messages []models.OutgoingMessage
	if err := db.Find(&messages).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
		t.Fatal(err)
	}

	if len(messages) != 1 {
		t.Error("Db update failed to roll back.")
	}

	err = sdb.Update(func(tx *gorm.DB)error {
		err := errors.New("atomic update failure")

		if err := tx.Save(&models.OutgoingMessage{ID:"abc"}).Error; err != nil {
			t.Fatal(err)
		}
		return err
	})
	if err == nil {
		t.Error("Update function did not return error")
	}

	var messages2 []models.OutgoingMessage
	if err := db.Find(&messages2).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
		t.Fatal(err)
	}

	if len(messages) > 1 {
		t.Error("Db update failed to roll back.")
	}
}

func TestSqliteDB_View(t *testing.T) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sdb := &SqliteDB{db, sync.RWMutex{}}

	if err := autoMigrateDatabase(db); err != nil {
		t.Fatal(err)
	}

	err = sdb.Update(func(tx *gorm.DB)error {
		return tx.Save(&models.OutgoingMessage{ID:"abc"}).Error
	})
	if err != nil {
		t.Error(err)
	}

	var messages []models.OutgoingMessage
	err = sdb.View(func(tx *gorm.DB)error {
		return tx.Find(&messages).Error
	})
	if err != nil {
		t.Error(err)
	}
	if len(messages) != 1 {
		t.Error("Failed to return messages")
	}
}

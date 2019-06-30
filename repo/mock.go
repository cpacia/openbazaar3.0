package repo

import (
	"github.com/jinzhu/gorm"
	"sync"
)

// MockDB returns an in-memory sqlitdb.
func MockDB() (Database, error) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := autoMigrateDatabase(db); err != nil {
		return nil, err
	}
	return &SqliteDB{db, sync.RWMutex{}}, nil
}

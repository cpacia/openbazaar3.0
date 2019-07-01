package repo

import (
	"github.com/jinzhu/gorm"
	"math/rand"
	"os"
	"path"
	"strconv"
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

// MockRepo returns a repo which uses a tmp data directory
// and in-memory database.
func MockRepo() (*Repo, error) {
	n := rand.Intn(1000000)
	dataDir := path.Join(os.TempDir(), "openbazaar-test", strconv.Itoa(n))
	return newRepo(dataDir, "", true)
}

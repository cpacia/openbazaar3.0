package repo

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"math/rand"
	"os"
	"path"
	"strconv"
)

// MockDB returns an in-memory sqlite db.
func MockDB() (database.Database, error) {
	n := rand.Intn(1000000)
	dataDir := path.Join(os.TempDir(), "openbazaar-test", strconv.Itoa(n))
	db, err := ffsqlite.NewFFMemoryDB(dataDir)
	if err != nil {
		return nil, err
	}
	if err := autoMigrateDatabase(db); err != nil {
		return nil, err
	}
	return db, nil
}

// MockRepo returns a repo which uses a tmp data directory
// and in-memory database.
func MockRepo() (*Repo, error) {
	n := rand.Intn(1000000)
	dataDir := path.Join(os.TempDir(), "openbazaar-test", strconv.Itoa(n))
	return newRepo(dataDir, "", true)
}

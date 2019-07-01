package repo

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"path"
	"sync"
)

// SqliteDB is an implementation of the Database interface using
// the gorm ORM with sqlite.
type SqliteDB struct {
	db  *gorm.DB
	mtx sync.RWMutex
}

// NewSqliteDB instantiates a new db which satisfies the Database interface.
func NewSqliteDB(dataDir string) (*SqliteDB, error) {
	pth := path.Join(dataDir, "datastore", dbName)
	if dataDir == ":memory:" {
		pth = dataDir
	}
	db, err := gorm.Open("sqlite3", pth)
	if err != nil {
		return nil, err
	}
	return &SqliteDB{db, sync.RWMutex{}}, nil
}

// View invokes the passed function in the context of a managed
// read-only transaction.  Any errors returned from the user-supplied
// function are returned from this function.
//
// Calling Rollback or Commit on the transaction passed to the
// user-supplied function will result in a panic.
func (s *SqliteDB) View(fn func(tx *gorm.DB) error) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return fn(s.db)
}

// Update invokes the passed function in the context of a managed
// read-write transaction.  Any errors returned from the user-supplied
// function will cause the transaction to be rolled back and are
// returned from this function.  Otherwise, the transaction is committed
// when the user-supplied function returns a nil error.
//
// Calling Rollback or Commit on the transaction passed to the
// user-supplied function will result in a panic.
func (s *SqliteDB) Update(fn func(tx *gorm.DB) error) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	tx := s.db.Begin()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// Close cleanly shuts down the database and syncs all data.  It will
// block until all database transactions have been finalized (rolled
// back or committed).
func (s *SqliteDB) Close() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.db.Close()
}

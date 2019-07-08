package ffsqlite

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"os"
	"path"
	"sync"
)

const (
	dbName = "openbazaar.db"
)

// FFSqliteDB is an implementation of the Database interface using
// flat file store for the public data and a sqlite database.
type FFSqliteDB struct {
	db   *gorm.DB
	ffdb *FlatFileDB
	mtx  sync.RWMutex
}

// NewFFSqliteDB instantiates a new db which satisfies the Database interface.
func NewFFSqliteDB(dataDir string) (database.Database, error) {
	db, err := gorm.Open("sqlite3", path.Join(dataDir, "datastore", dbName))
	if err != nil {
		return nil, err
	}
	ffdb, err := NewFlatFileDB(path.Join(dataDir, "public"))
	if err != nil {
		return nil, err
	}
	return &FFSqliteDB{db: db, ffdb: ffdb, mtx: sync.RWMutex{}}, nil
}

// NewFFSqliteDB instantiates a new db which satisfies the Database interface.
// The sqlite db will be held in memory.
func NewFFMemoryDB(dataDir string) (database.Database, error) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	ffdb, err := NewFlatFileDB(path.Join(dataDir, "public"))
	if err != nil {
		return nil, err
	}
	return &FFSqliteDB{db: db, ffdb: ffdb, mtx: sync.RWMutex{}}, nil
}

// View invokes the passed function in the context of a managed
// read-only transaction.  Any errors returned from the user-supplied
// function are returned from this function.
//
// Calling Rollback or Commit on the transaction passed to the
// user-supplied function will result in a panic.
func (fdb *FFSqliteDB) View(fn func(tx database.Tx) error) error {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	tx := readTx(fdb.db, fdb.ffdb)
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Update invokes the passed function in the context of a managed
// read-write transaction.  Any errors returned from the user-supplied
// function will cause the transaction to be rolled back and are
// returned from this function.  Otherwise, the transaction is committed
// when the user-supplied function returns a nil error.
//
// Calling Rollback or Commit on the transaction passed to the
// user-supplied function will result in a panic.
func (fdb *FFSqliteDB) Update(fn func(tx database.Tx) error) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	tx := writeTx(fdb.db, fdb.ffdb)
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// PublicDataPath returns the path to the public data directory.
func (fdb *FFSqliteDB) PublicDataPath() string {
	return fdb.ffdb.Path()
}

// Close cleanly shuts down the database and syncs all data.  It will
// block until all database transactions have been finalized (rolled
// back or committed).
func (fdb *FFSqliteDB) Close() error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	return fdb.db.Close()
}

type tx struct {
	dbtx *gorm.DB
	ffdb *FlatFileDB

	rollbackCache []interface{}
	commitCache   []interface{}

	closed      bool
	isForWrites bool
}

type deleteListing string

func writeTx(db *gorm.DB, ffdb *FlatFileDB) database.Tx {
	dbtx := db.Begin()
	return &tx{dbtx: dbtx, ffdb: ffdb, isForWrites: true}
}

func readTx(db *gorm.DB, ffdb *FlatFileDB) database.Tx {
	return &tx{dbtx: db, ffdb: ffdb, isForWrites: false}
}

func (t *tx) Commit() error {
	if t.closed {
		panic("tx already closed")
	}

	defer func() { t.closed = true }()

	if !t.isForWrites {
		return nil
	}

	for _, i := range t.commitCache {
		if err := t.setInterfaceType(i); err != nil {
			t.Rollback()
			return err
		}
	}

	if err := t.dbtx.Commit().Error; err != nil {
		t.Rollback()
		return err
	}
	return nil
}

func (t *tx) Rollback() error {
	if t.closed {
		panic("tx already closed")
	}

	defer func() { t.closed = true }()

	if !t.isForWrites {
		return nil
	}

	for _, i := range t.rollbackCache {
		if err := t.setInterfaceType(i); err != nil {
			return err
		}
	}

	if err := t.dbtx.Rollback().Error; err != nil {
		return err
	}
	return nil
}

func (t *tx) DB() *gorm.DB {
	return t.dbtx
}

func (t *tx) GetProfile() (*models.Profile, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		profile, ok := t.commitCache[x].(*models.Profile)
		if ok {
			return profile, nil
		}
	}
	return t.ffdb.GetProfile()
}

func (t *tx) SetProfile(profile *models.Profile) error {
	current, err := t.ffdb.GetProfile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, profile)
	return nil
}

func (t *tx) GetFollowers() (models.Followers, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		followers, ok := t.commitCache[x].(models.Followers)
		if ok {
			return followers, nil
		}
	}
	return t.ffdb.GetFollowers()
}

func (t *tx) SetFollowers(followers models.Followers) error {
	current, err := t.ffdb.GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, followers)
	return nil
}

func (t *tx) GetFollowing() (models.Following, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		following, ok := t.commitCache[x].(models.Following)
		if ok {
			return following, nil
		}
	}
	return t.ffdb.GetFollowing()
}

func (t *tx) SetFollowing(following models.Following) error {
	current, err := t.ffdb.GetFollowing()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, following)
	return nil
}

func (t *tx) GetListing(slug string) (*pb.Listing, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		listing, ok := t.commitCache[x].(*pb.SignedListing)
		if ok && listing.Listing.Slug == slug {
			return listing.Listing, nil
		}
	}
	return t.ffdb.GetListing(slug)
}

func (t *tx) SetListing(listing *pb.SignedListing) error {
	current, err := t.ffdb.getSignedListing(listing.Listing.Slug)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, listing)
	return nil
}

func (t *tx) DeleteListing(slug string) error {
	current, err := t.ffdb.getSignedListing(slug)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, deleteListing(slug))
	return nil
}

func (t *tx) GetListingIndex() (models.ListingIndex, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		index, ok := t.commitCache[x].(models.ListingIndex)
		if ok {
			return index, nil
		}
	}
	return t.ffdb.GetListingIndex()
}

func (t *tx) SetListingIndex(index models.ListingIndex) error {
	current, err := t.ffdb.GetListingIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, index)
	return nil
}

func (t *tx) setInterfaceType(i interface{}) error {
	switch i.(type) {
	case *models.Profile:
		if i.(*models.Profile) == nil {
			return nil
		}
		if err := t.ffdb.SetProfile(i.(*models.Profile)); err != nil {
			return err
		}
	case models.Followers:
		if err := t.ffdb.SetFollowers(i.(models.Followers)); err != nil {
			return err
		}
	case models.Following:
		if err := t.ffdb.SetFollowing(i.(models.Following)); err != nil {
			return err
		}
	case *pb.SignedListing:
		if i.(*pb.SignedListing) == nil {
			return nil
		}
		if err := t.ffdb.SetListing(i.(*pb.SignedListing)); err != nil {
			return err
		}
	case models.ListingIndex:
		if err := t.ffdb.SetListingIndex(i.(models.ListingIndex)); err != nil {
			return err
		}
	case deleteListing:
		if err := t.ffdb.DeleteListing(string(i.(deleteListing))); err != nil {
			return err
		}
	}
	return nil
}

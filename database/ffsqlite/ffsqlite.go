package ffsqlite

import (
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Import sqlite dialect
	"os"
	"path"
	"sync"
)

const (
	dbName = "openbazaar.db"
)

var ErrReadOnly = errors.New("tx is read only")

// FFSqliteDB is an implementation of the Database interface using
// flat file store for the public data and a sqlite database.
type DB struct {
	db   *gorm.DB
	ffdb *FlatFileDB
	mtx  sync.Mutex
}

// NewFFSqliteDB instantiates a new db which satisfies the Database interface.
func NewFFSqliteDB(dataDir string) (database.Database, error) {
	db, err := gorm.Open("sqlite3", path.Join(dataDir, dbName))
	if err != nil {
		return nil, err
	}
	ffdb, err := NewFlatFileDB(path.Join(dataDir, "public"))
	if err != nil {
		return nil, err
	}
	return &DB{db: db, ffdb: ffdb, mtx: sync.Mutex{}}, nil
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
	return &DB{db: db, ffdb: ffdb, mtx: sync.Mutex{}}, nil
}

// View invokes the passed function in the context of a managed
// read-only transaction.  Any errors returned from the user-supplied
// function are returned from this function.
//
// Calling Rollback or Commit on the transaction passed to the
// user-supplied function will result in a panic.
func (fdb *DB) View(fn func(tx database.Tx) error) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

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
func (fdb *DB) Update(fn func(tx database.Tx) error) error {
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
func (fdb *DB) PublicDataPath() string {
	return fdb.ffdb.Path()
}

// Close cleanly shuts down the database and syncs all data.  It will
// block until all database transactions have been finalized (rolled
// back or committed).
func (fdb *DB) Close() error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	return fdb.db.Close()
}

type tx struct {
	dbtx *gorm.DB
	ffdb *FlatFileDB

	rollbackCache []interface{}
	commitCache   []interface{}

	commitHooks []func()

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

// Commit commits all changes that have been made to the db or public data.
// Depending on the backend implementation this could be to a cache that
// is periodically synced to persistent storage or directly to persistent
// storage.  In any case, all transactions which are started after the commit
// finishes will include all changes made by this transaction.  Calling this
// function on a managed transaction will result in a panic.
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
	for _, fn := range t.commitHooks {
		fn()
	}
	return nil
}

// Rollback undoes all changes that have been made to the db or public
// data.  Calling this function on a managed transaction will result in
// a panic.
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

// Save will save the passed in model to the database. If it already exists
// it will be overridden.
func (t *tx) Save(model interface{}) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	return t.dbtx.Save(model).Error
}

// Read returns the underlying sql database in a read-only mode so that
// queries can be made against it.
func (t *tx) Read() *gorm.DB {
	return t.dbtx
}

// Update will update the given key to the value for the given model. The
// where map can be used to impose extra conditions on which specific model
// gets updated. The map key must be of the format "key = ?". This allows
// for using alternative conditions such as "timestamp <= ?".
func (t *tx) Update(key string, value interface{}, where map[string]interface{}, model interface{}) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	db := t.dbtx.Model(model)
	for k, v := range where {
		db = db.Where(k, v)
	}
	return db.UpdateColumn(key, value).Error
}

// Delete will delete all models of the given type from the database where
// field == key.
func (t *tx) Delete(key string, value interface{}, where map[string]interface{}, model interface{}) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	db := t.dbtx.Model(model)
	for k, v := range where {
		db = db.Where(k, v)
	}
	return db.Where(fmt.Sprintf("%s = ?", key), value).Delete(model).Error
}

// Migrate will auto-migrate the database to from any previous schema for this
// model to the current schema.
func (t *tx) Migrate(model interface{}) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	return t.dbtx.AutoMigrate(model).Error
}

// RegisterCommitHook registers a callback that is invoked whenever a commit completes
// successfully.
func (t *tx) RegisterCommitHook(fn func()) {
	t.commitHooks = append(t.commitHooks, fn)
}

// GetProfile returns the profile.
func (t *tx) GetProfile() (*models.Profile, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		profile, ok := t.commitCache[x].(*models.Profile)
		if ok {
			return profile, nil
		}
	}
	return t.ffdb.GetProfile()
}

// SetProfile sets the profile.
func (t *tx) SetProfile(profile *models.Profile) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.GetProfile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, profile)
	return nil
}

// GetFollowers returns followers list.
func (t *tx) GetFollowers() (models.Followers, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		followers, ok := t.commitCache[x].(models.Followers)
		if ok {
			return followers, nil
		}
	}
	return t.ffdb.GetFollowers()
}

// SetFollowers sets the followers list.
func (t *tx) SetFollowers(followers models.Followers) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, followers)
	return nil
}

// GetFollowing returns the following list.
func (t *tx) GetFollowing() (models.Following, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		following, ok := t.commitCache[x].(models.Following)
		if ok {
			return following, nil
		}
	}
	return t.ffdb.GetFollowing()
}

// SetFollowing sets the following list.
func (t *tx) SetFollowing(following models.Following) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.GetFollowing()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, following)
	return nil
}

// GetListing returns the listing for the given slug.
func (t *tx) GetListing(slug string) (*pb.SignedListing, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		listing, ok := t.commitCache[x].(*pb.SignedListing)
		if ok && listing.Listing.Slug == slug {
			return listing, nil
		}
	}
	return t.ffdb.GetListing(slug)
}

// SetListing saves the given listing.
func (t *tx) SetListing(listing *pb.SignedListing) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.getSignedListing(listing.Listing.Slug)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, listing)
	return nil
}

// DeleteListing deletes the given listing.
func (t *tx) DeleteListing(slug string) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.getSignedListing(slug)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, deleteListing(slug))
	return nil
}

// GetListingIndex returns the listing index.
func (t *tx) GetListingIndex() (models.ListingIndex, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		index, ok := t.commitCache[x].(models.ListingIndex)
		if ok {
			return index, nil
		}
	}
	return t.ffdb.GetListingIndex()
}

// SetListingIndex sets the listing index.
func (t *tx) SetListingIndex(index models.ListingIndex) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.GetListingIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, index)
	return nil
}

// GetRatingIndex returns the rating index.
func (t *tx) GetRatingIndex() (models.RatingIndex, error) {
	for x := len(t.commitCache) - 1; x >= 0; x-- {
		index, ok := t.commitCache[x].(models.RatingIndex)
		if ok {
			return index, nil
		}
	}
	return t.ffdb.GetRatingIndex()
}

// SetRatingIndex sets the rating index.
func (t *tx) SetRatingIndex(index models.RatingIndex) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	current, err := t.ffdb.GetRatingIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	t.rollbackCache = append(t.rollbackCache, current)
	t.commitCache = append(t.commitCache, index)
	return nil
}

// SetRating saves the given rating.
func (t *tx) SetRating(rating *pb.Rating) error {
	if !t.isForWrites {
		return ErrReadOnly
	}
	t.commitCache = append(t.commitCache, rating)
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
	case models.RatingIndex:
		if err := t.ffdb.SetRatingIndex(i.(models.RatingIndex)); err != nil {
			return err
		}
	case *pb.Rating:
		if i.(*pb.Rating) == nil {
			return nil
		}
		if err := t.ffdb.SetRating(i.(*pb.Rating)); err != nil {
			return err
		}
	case deleteListing:
		if err := t.ffdb.DeleteListing(string(i.(deleteListing))); err != nil {
			return err
		}
	}
	return nil
}

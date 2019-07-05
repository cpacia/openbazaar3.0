package database

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/jinzhu/gorm"
)

// PublicData is the interface for access to the node's IPFS public
// data directory. This data is visible by other nodes on the network.
type PublicData interface {
	// GetProfile returns the profile.
	GetProfile() (*models.Profile, error)

	// SetProfile sets the profile.
	SetProfile(profile *models.Profile) error

	// GetFollowers returns followers list.
	GetFollowers() (models.Followers, error)

	// SetFollowers sets the followers list.
	SetFollowers(followers models.Followers) error

	// GetFollowing returns the following list.
	GetFollowing() (models.Following, error)

	// SetFollowing sets the following list.
	SetFollowing(following models.Following) error

	// GetListing returns the listing for the given slug.
	GetListing(slug string) (*pb.Listing, error)

	// SetListing saves the given listing.
	SetListing(listing *pb.SignedListing) error

	// DeleteListing deletes the given listing.
	DeleteListing(slug string) error

	// GetListingIndex returns the listing index.
	GetListingIndex() (models.ListingIndex, error)

	// SetListingIndex sets the listing index.
	SetListingIndex(index models.ListingIndex) error
}

// Tx represents a database transaction.  It can either by read-only or
// read-write.  The transaction provides access to a sql database interface
// with an open transaction to use for writing generic data.
// It also provides methods for reading and writing the node's public data.
//
// As would be expected with a transaction, no changes will be saved to the
// database until it has been committed.  The transaction will only provide a
// view of the database at the time it was created.  Transactions should not be
// long running operations.
//
// Public data methods may return an os.IsNotFound error if the data is not found.
type Tx interface {
	// Commit commits all changes that have been made to the db or public data.
	// Depending on the backend implementation this could be to a cache that
	// is periodically synced to persistent storage or directly to persistent
	// storage.  In any case, all transactions which are started after the commit
	// finishes will include all changes made by this transaction.  Calling this
	// function on a managed transaction will result in a panic.
	Commit() error

	// Rollback undoes all changes that have been made to the db or public
	// data.  Calling this function on a managed transaction will result in
	// a panic.
	Rollback() error

	// DB returns the underlying sql database. A transaction
	// should be open on the db and any changes should be atomic.
	DB() *gorm.DB

	// PublicData provides atomic access to the IPFS data directory.
	PublicData
}

// Database is an interface which exposes a minimal amount of functions methods
// needed to atomically read and write to the database.
type Database interface {
	// View invokes the passed function in the context of a managed
	// read-only transaction.  Any errors returned from the user-supplied
	// function are returned from this function.
	//
	// Calling Rollback or Commit on the transaction passed to the
	// user-supplied function will result in a panic.
	View(fn func(tx Tx) error) error

	// Update invokes the passed function in the context of a managed
	// read-write transaction.  Any errors returned from the user-supplied
	// function will cause the transaction to be rolled back and are
	// returned from this function.  Otherwise, the transaction is committed
	// when the user-supplied function returns a nil error.
	//
	// Calling Rollback or Commit on the transaction passed to the
	// user-supplied function will result in a panic.
	Update(fn func(tx Tx) error) error

	// PublicDataPath returns the path to the public data directory.
	PublicDataPath() string

	// Close cleanly shuts down the database and syncs all data.  It will
	// block until all database transactions have been finalized (rolled
	// back or committed).
	Close() error
}

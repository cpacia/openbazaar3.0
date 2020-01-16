package ffsqlite

import (
	"encoding/json"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

const (
	// ProfileFile is the filename of the profile on disk.
	ProfileFile = "profile.json"
	// FollowersFile is the filename of the followers file on disk.
	FollowersFile = "followers.json"
	// FollowingFile is the filename of the following file on disk.
	FollowingFile = "following.json"
	// ListingIndexFile is the filename of the listing index file on disk.
	ListingIndexFile = "listings.json"
	// RatingIndexFile is the filename of the rating index file on disk.
	RatingIndexFile = "ratings.json"
)

// FlatFileDB represents the IPFS root directory that holds the node's
// public data. This includes things like the profile and listings.
// This object will maintain consistency by updating all pieces whenever
// changes are made. For example, updating a listing will also update
// the listing index.
type FlatFileDB struct {
	rootDir string

	mtx sync.RWMutex
}

// NewFlatFileDB returns a new public data directory. If one does not
// already exist at the given location, it will be initialized.
func NewFlatFileDB(rootDir string) (*FlatFileDB, error) {
	fdb := &FlatFileDB{rootDir, sync.RWMutex{}}

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := fdb.initializeDirectory(); err != nil {
			return nil, err
		}
	}

	return fdb, nil
}

// Path returns the path to the public directory.
func (fdb *FlatFileDB) Path() string {
	return fdb.rootDir
}

// GetProfile loads the profile from disk and returns it.
func (fdb *FlatFileDB) GetProfile() (*models.Profile, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, ProfileFile))
	if err != nil {
		return nil, err
	}
	profile := new(models.Profile)
	err = json.Unmarshal(raw, profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// SetProfile saves the profile to disk.
func (fdb *FlatFileDB) SetProfile(profile *models.Profile) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	out, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, ProfileFile), out, os.ModePerm)
}

// GetFollowers loads the follower list from disk and returns it.
func (fdb *FlatFileDB) GetFollowers() (models.Followers, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, FollowersFile))
	if err != nil {
		return nil, err
	}
	var followers models.Followers
	err = json.Unmarshal(raw, &followers)
	if err != nil {
		return nil, err
	}
	return followers, nil
}

// SetFollowers saves the followers list to disk.
func (fdb *FlatFileDB) SetFollowers(followers models.Followers) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	out, err := json.MarshalIndent(followers, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, FollowersFile), out, os.ModePerm)
}

// GetFollowing loads the following list from disk and returns it.
func (fdb *FlatFileDB) GetFollowing() (models.Following, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, FollowingFile))
	if err != nil {
		return nil, err
	}
	var following models.Following
	err = json.Unmarshal(raw, &following)
	if err != nil {
		return nil, err
	}
	return following, nil
}

// SetFollowing saves the following list to disk.
func (fdb *FlatFileDB) SetFollowing(following models.Following) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	out, err := json.MarshalIndent(following, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, FollowingFile), out, os.ModePerm)
}

// GetListing loads the listing from disk and returns it.
func (fdb *FlatFileDB) GetListing(slug string) (*pb.SignedListing, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, "listings", slug+".json"))
	if err != nil {
		return nil, err
	}

	var sl pb.SignedListing
	err = jsonpb.UnmarshalString(string(raw), &sl)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// getSignedListing loads the full signed listing from disk and returns it.
func (fdb *FlatFileDB) getSignedListing(slug string) (*pb.SignedListing, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, "listings", slug+".json"))
	if err != nil {
		return nil, err
	}

	var sl pb.SignedListing
	err = jsonpb.UnmarshalString(string(raw), &sl)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// SetListing saves the listing to disk.
func (fdb *FlatFileDB) SetListing(listing *pb.SignedListing) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	m := jsonpb.Marshaler{
		EmitDefaults: false,
		Indent:       "    ",
	}
	out, err := m.MarshalToString(listing)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, "listings", listing.Listing.Slug+".json"), []byte(out), os.ModePerm)
}

// DeleteListing deletes a listing from disk given the slug.
func (fdb *FlatFileDB) DeleteListing(slug string) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	return os.Remove(path.Join(fdb.rootDir, "listings", slug+".json"))
}

// GetListingIndex loads the listing index from disk and returns it.
func (fdb *FlatFileDB) GetListingIndex() (models.ListingIndex, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, ListingIndexFile))
	if err != nil {
		return nil, err
	}
	var index models.ListingIndex
	err = json.Unmarshal(raw, &index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

// SetListingIndex saves the listing index to disk.
func (fdb *FlatFileDB) SetListingIndex(index models.ListingIndex) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	out, err := json.MarshalIndent(index, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, RatingIndexFile), out, os.ModePerm)
}

// GetListingIndex returns the rating index.
func (fdb *FlatFileDB) GetRatingIndex() (models.RatingIndex, error) {
	fdb.mtx.RLock()
	defer fdb.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(fdb.rootDir, RatingIndexFile))
	if err != nil {
		return nil, err
	}
	var index models.RatingIndex
	err = json.Unmarshal(raw, &index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

// SetRatingIndex sets the rating index.
func (fdb *FlatFileDB) SetRatingIndex(index models.RatingIndex) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	out, err := json.MarshalIndent(index, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, RatingIndexFile), out, os.ModePerm)
}

// SetRating saves the given rating.
func (fdb *FlatFileDB) SetRating(rating *pb.Rating) error {
	fdb.mtx.Lock()
	defer fdb.mtx.Unlock()

	ser, err := proto.Marshal(rating)
	if err != nil {
		return err
	}
	h, err := utils.MultihashSha256(ser)
	if err != nil {
		return err
	}

	m := jsonpb.Marshaler{
		EmitDefaults: false,
		Indent:       "    ",
	}
	out, err := m.MarshalToString(rating)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(fdb.rootDir, "ratings", h.B58String()[:16]+".json"), []byte(out), os.ModePerm)
}

// dataPathJoin is a helper function which joins the pathArgs to the service's
// dataPath and returns the result
func (fdb *FlatFileDB) dataPathJoin(pathArgs ...string) string {
	allPathArgs := append([]string{fdb.rootDir}, pathArgs...)
	return filepath.Join(allPathArgs...)
}

func (fdb *FlatFileDB) initializeDirectory() error {
	directories := []string{
		fdb.rootDir,
		fdb.dataPathJoin("listings"),
		fdb.dataPathJoin("ratings"),
		fdb.dataPathJoin("images"),
		fdb.dataPathJoin("images", "tiny"),
		fdb.dataPathJoin("images", "small"),
		fdb.dataPathJoin("images", "medium"),
		fdb.dataPathJoin("images", "large"),
		fdb.dataPathJoin("images", "original"),
		fdb.dataPathJoin("posts"),
		fdb.dataPathJoin("files"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

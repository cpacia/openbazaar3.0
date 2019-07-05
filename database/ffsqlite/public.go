package ffsqlite

import (
	"encoding/json"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
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
)

// PublicData represents the IPFS root directory that holds the node's
// public data. This includes things like the profile and listings.
// This object will maintain consistency by updating all pieces whenever
// changes are made. For example, updating a listing will also update
// the listing index.
type PublicData struct {
	rootDir string

	mtx sync.RWMutex
}

// NewPublicData returns a new public data directory. If one does not
// already exist at the given location, it will be initialized.
func NewPublicData(rootDir string) (*PublicData, error) {
	pd := &PublicData{rootDir, sync.RWMutex{}}

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := pd.initializeDirectory(); err != nil {
			return nil, err
		}
	}

	return pd, nil
}

// Path returns the path to the public directory.
func (pd *PublicData) Path() string {
	return pd.rootDir
}

// GetProfile loads the profile from disk and returns it.
func (pd *PublicData) GetProfile() (*models.Profile, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, ProfileFile))
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
func (pd *PublicData) SetProfile(profile *models.Profile) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	out, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, ProfileFile), out, os.ModePerm)
}

// GetFollowers loads the follower list from disk and returns it.
func (pd *PublicData) GetFollowers() (models.Followers, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, FollowersFile))
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
func (pd *PublicData) SetFollowers(followers models.Followers) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	out, err := json.MarshalIndent(followers, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, FollowersFile), out, os.ModePerm)
}

// GetFollowing loads the following list from disk and returns it.
func (pd *PublicData) GetFollowing() (models.Following, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, FollowingFile))
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
func (pd *PublicData) SetFollowing(following models.Following) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	out, err := json.MarshalIndent(following, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, FollowingFile), out, os.ModePerm)
}

// GetListing loads the listing from disk and returns it.
func (pd *PublicData) GetListing(slug string) (*pb.Listing, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, "listings", slug+".json"))
	if err != nil {
		return nil, err
	}

	var sl pb.SignedListing
	err = jsonpb.UnmarshalString(string(raw), &sl)
	if err != nil {
		return nil, err
	}
	return sl.Listing, nil
}

// getSignedListing loads the full signed listing from disk and returns it.
func (pd *PublicData) getSignedListing(slug string) (*pb.SignedListing, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, "listings", slug+".json"))
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
func (pd *PublicData) SetListing(listing *pb.SignedListing) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	m := jsonpb.Marshaler{
		EmitDefaults: false,
		Indent:       "    ",
	}
	out, err := m.MarshalToString(listing)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, "listings", listing.Listing.Slug+".json"), []byte(out), os.ModePerm)
}

// DeleteListing deletes a listing from disk given the slug.
func (pd *PublicData) DeleteListing(slug string) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	return os.Remove(path.Join(pd.rootDir, "listings", slug+".json"))
}

// GetListingIndex loads the listing index from disk and returns it.
func (pd *PublicData) GetListingIndex() (models.ListingIndex, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, ListingIndexFile))
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
func (pd *PublicData) SetListingIndex(index models.ListingIndex) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	out, err := json.MarshalIndent(index, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, ListingIndexFile), out, os.ModePerm)
}

// dataPathJoin is a helper function which joins the pathArgs to the service's
// dataPath and returns the result
func (pd *PublicData) dataPathJoin(pathArgs ...string) string {
	allPathArgs := append([]string{pd.rootDir}, pathArgs...)
	return filepath.Join(allPathArgs...)
}

func (pd *PublicData) initializeDirectory() error {
	directories := []string{
		pd.rootDir,
		pd.dataPathJoin("listings"),
		pd.dataPathJoin("ratings"),
		pd.dataPathJoin("images"),
		pd.dataPathJoin("images", "tiny"),
		pd.dataPathJoin("images", "small"),
		pd.dataPathJoin("images", "medium"),
		pd.dataPathJoin("images", "large"),
		pd.dataPathJoin("images", "original"),
		pd.dataPathJoin("posts"),
		pd.dataPathJoin("channel"),
		pd.dataPathJoin("files"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

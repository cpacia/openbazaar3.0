package repo

import (
	"context"
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gogo/protobuf/proto"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/namesys"
	ipnspb "github.com/ipfs/go-ipns/pb"
	fpath "github.com/ipfs/go-path"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

const (
	profileFile   = "profile.json"
	followersFile = "followers.json"
	followingFile = "following.json"
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

// GetProfile loads the profile from disk and returns it.
func (pd *PublicData) GetProfile() (*models.Profile, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, profileFile))
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
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	out, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, profileFile), out, os.ModePerm)
}

// GetFollowers loads the follower list from disk and returns it.
func (pd *PublicData) GetFollowers() (models.Followers, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, followersFile))
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
func (pd *PublicData) SetFollowers(followers *models.Followers) error {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	out, err := json.MarshalIndent(followers, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, followersFile), out, os.ModePerm)
}

// GetFollowing loads the following list from disk and returns it.
func (pd *PublicData) GetFollowing() (models.Following, error) {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	raw, err := ioutil.ReadFile(path.Join(pd.rootDir, followingFile))
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
func (pd *PublicData) SetFollowing(following *models.Following) error {
	pd.mtx.RLock()
	defer pd.mtx.RUnlock()

	out, err := json.MarshalIndent(following, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(pd.rootDir, followingFile), out, os.ModePerm)
}

// Publish does the following:
// - Recursively unpin the current root directory. This will allow old objects to be garbage collected.
// - Add and pin the new directory and all files and subdirectories inside.
// - Publish the new root to IPNS.
func (pd *PublicData) Publish(ctx context.Context, ipfsNode *core.IpfsNode) error {
	pd.mtx.Lock()
	defer pd.mtx.Unlock()

	api, err := coreapi.NewCoreAPI(ipfsNode)
	if err != nil {
		return err
	}

	currentRoot := currentRootHash(ipfsNode)

	// First uppin old root hash
	if currentRoot != "" {
		rp, err := api.ResolvePath(context.Background(), ipath.New(currentRoot))
		if err != nil {
			return err
		}

		if err := api.Pin().Rm(context.Background(), rp, options.Pin.RmRecursive(true)); err != nil {
			return err
		}
	}

	// Add the directory to IPFS
	stat, err := os.Lstat(pd.rootDir)
	if err != nil {
		return err
	}

	f, err := files.NewSerialFile(pd.rootDir, false, stat)
	if err != nil {
		return err
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(true),
	}
	pth, err := api.Unixfs().Add(context.Background(), files.ToDir(f), opts...)
	if err != nil {
		return err
	}

	// Publish
	return ipfsNode.Namesys.Publish(ctx, ipfsNode.PrivateKey, fpath.FromString(pth.Root().String()))
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

func currentRootHash(nd *core.IpfsNode) string {
	ipnskey := namesys.IpnsDsKey(nd.Identity)
	ival, err := nd.Repo.Datastore().Get(ipnskey)
	if err != nil {
		return ""
	}

	ourIpnsRecord := new(ipnspb.IpnsEntry)
	err = proto.Unmarshal(ival, ourIpnsRecord)
	if err != nil {
		// If this cannot be unmarhsaled due to an error we should
		// delete the key so that it doesn't cause other processes to
		// fail. The publisher will re-create a new one.
		nd.Repo.Datastore().Delete(ipnskey)
		return ""
	}
	return string(ourIpnsRecord.Value)
}

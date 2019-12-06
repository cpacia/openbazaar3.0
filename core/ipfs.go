package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/coreunix"
	"github.com/ipfs/go-ipfs/namesys"
	ipnspb "github.com/ipfs/go-ipns/pb"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/interface-go-ipfs-core/options"
	nameopts "github.com/ipfs/interface-go-ipfs-core/options/namesys"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"os"
	gopath "path"
	"strings"
	"sync"
	"time"
)

const (
	catTimeout     = time.Second * 30
	resolveTimeout = time.Second * 120
)

// cat fetches a file from IPFS given a path.
func (n *OpenBazaarNode) cat(ctx context.Context, pth path.Path) ([]byte, error) {
	catDone := make(chan struct{})
	ctx, cancel := context.WithTimeout(ctx, catTimeout)
	defer func() {
		cancel()
		catDone <- struct{}{}
	}()

	capi, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-catDone:
			return
		case <-n.shutdown:
			cancel()
		}
	}()

	nd, err := capi.Unixfs().Get(ctx, pth)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}

	r, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("incorrect type from Unixfs().Get()")
	}

	return ioutil.ReadAll(r)
}

// add imports the given file into ipfs and returns the cid.
func (n *OpenBazaarNode) add(ctx context.Context, filePath string) (cid.Cid, error) {
	defer n.ipfsNode.Blockstore.PinLock().Unlock()

	stat, err := os.Lstat(filePath)
	if err != nil {
		return cid.Cid{}, err
	}

	f, err := files.NewSerialFile(filePath, false, stat)
	if err != nil {
		return cid.Cid{}, err
	}
	defer f.Close()

	fileAdder, err := coreunix.NewAdder(ctx, n.ipfsNode.Pinning, n.ipfsNode.Blockstore, n.ipfsNode.DAG)
	if err != nil {
		return cid.Cid{}, err
	}
	node, err := fileAdder.AddAllAndPin(f)
	if err != nil {
		return cid.Cid{}, err
	}
	return node.Cid(), nil
}

// cid returns the content ID of the byte array. Note care must be taken when using this
// as of now it seems the only IPFS API I can find requires adding the file to get the cid.
// Thus we unpin the file after adding so it doesn't persist forever. If this file was
// supposed to be pinning then this function may unintentionally unpin it. Where we use
// this function currently this is OK since we do a publish immediately after which would
// re-pin any unpin objects in the data directory.
func (n *OpenBazaarNode) cid(file []byte) (cid.Cid, error) {
	api, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return cid.Cid{}, err
	}
	b := make([]byte, 20)
	rand.Read(b)
	dir := gopath.Join(os.TempDir(), "openbazaar-files")

	os.MkdirAll(dir, os.ModePerm)

	pth := gopath.Join(dir, hex.EncodeToString(b))
	defer os.Remove(pth)

	if err := ioutil.WriteFile(pth, file, os.ModePerm); err != nil {
		return cid.Cid{}, err
	}

	cid, err := n.add(context.Background(), pth)
	if err != nil {
		return cid, err
	}

	rp, err := api.ResolvePath(context.Background(), path.IpfsPath(cid))
	if err != nil {
		return cid, err
	}

	return cid, api.Pin().Rm(context.Background(), rp, options.Pin.RmRecursive(true))
}

// pin fetches a file from IPFS given a path and pins it.
func (n *OpenBazaarNode) pin(ctx context.Context, pth path.Path) error {
	pinDone := make(chan struct{})
	ctx, cancel := context.WithTimeout(ctx, catTimeout)
	defer func() {
		cancel()
		pinDone <- struct{}{}
	}()

	api, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-pinDone:
			return
		case <-n.shutdown:
			cancel()
		}
	}()

	return api.Pin().Add(ctx, pth)
}

// resolve an IPNS record. This is a multi-step process.
// If the usecache flag is provided we will attempt to load the record from the database. If
// it succeeds we will update the cache in a separate goroutine.
//
// If we need to actually get a record from the network the IPNS namesystem will first check to
// see if it is subscribed to the name with pubsub. If so, it will return cache from the database.
// If not, it will subscribe to the name and proceed to a DHT query to find the record.
// If the DHT query returns nothing it will finally attempt to return from cache.
// All subsequent resolves will return from cache as the pubsub will update the cache in real time
// as new records are published.
func (n *OpenBazaarNode) resolve(ctx context.Context, p peer.ID, usecache bool) (path.Path, error) {
	if usecache {
		var pth path.Path
		err := n.repo.DB().View(func(tx database.Tx) error {
			var err error
			pth, err = getFromDatastore(tx, p)
			return err
		})
		if err == nil {
			// Update the cache in background
			go func() {
				pth, err := n.resolveOnce(context.Background(), p, resolveTimeout, n.ipnsQuorum)
				if err != nil {
					return
				}
				if n.Identity() != p {
					err = n.repo.DB().Update(func(tx database.Tx) error {
						return putToDatastoreCache(tx, p, pth)
					})
					if err != nil {
						log.Error("Error putting IPNS record to datastore: %s", err.Error())
					}

				}
			}()
			return pth, nil
		}
	}
	pth, err := n.resolveOnce(ctx, p, resolveTimeout, n.ipnsQuorum)
	if err != nil {
		// Resolving fail. See if we have it in the db.
		var pth path.Path
		err := n.repo.DB().View(func(tx database.Tx) error {
			var err error
			pth, err = getFromDatastore(tx, p)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
		}
		return pth, nil
	}
	// Resolving succeeded. Update the cache.
	if n.Identity() != p {
		err = n.repo.DB().Update(func(tx database.Tx) error {
			return putToDatastoreCache(tx, p, pth)
		})
		if err != nil {
			log.Error("Error putting IPNS record to datastore: %s", err.Error())
		}
	}
	return pth, nil
}

func (n *OpenBazaarNode) resolveOnce(ctx context.Context, p peer.ID, timeout time.Duration, quorum uint) (path.Path, error) {
	resolveDone := make(chan struct{})
	defer func() {
		resolveDone <- struct{}{}
	}()
	ctx, cancel := context.WithTimeout(ctx, timeout)

	go func() {
		select {
		case <-resolveDone:
			return
		case <-n.shutdown:
			cancel()
		}
	}()

	pth, err := n.ipfsNode.Namesys.Resolve(ctx, "/ipns/"+p.Pretty(), nameopts.DhtRecordCount(quorum))
	if err != nil {
		return nil, err
	}
	return path.New(pth.String()), nil
}

// fetchGraph returns a list of CIDs in the root directory.
func (n *OpenBazaarNode) fetchGraph(ctx context.Context) ([]cid.Cid, error) {
	id, err := n.ipnsRecordValue()
	if err != nil {
		return nil, err
	}
	dag := merkledag.NewDAGService(n.ipfsNode.Blocks)
	var ret []cid.Cid
	l := new(sync.Mutex)
	m := make(map[string]bool)
	m[id.String()] = true
	for {
		if len(m) == 0 {
			break
		}
		for k := range m {
			c, err := cid.Decode(k)
			if err != nil {
				return ret, err
			}
			ret = append(ret, c)
			links, err := dag.GetLinks(ctx, c)
			if err != nil {
				return ret, err
			}
			l.Lock()
			delete(m, k)
			for _, link := range links {
				m[link.Cid.String()] = true
			}
			l.Unlock()
		}
	}
	return ret, nil
}

// ipfsRecordValue returns the current value of our ipns record.
func (n *OpenBazaarNode) ipnsRecordValue() (cid.Cid, error) {
	ipnskey := namesys.IpnsDsKey(n.Identity())
	ival, err := n.ipfsNode.Repo.Datastore().Get(ipnskey)
	if err != nil {
		return cid.Cid{}, err
	}

	ourIpnsRecord := new(ipnspb.IpnsEntry)
	err = proto.Unmarshal(ival, ourIpnsRecord)
	if err != nil {
		// If this cannot be unmarhsaled due to an error we should
		// delete the key so that it doesn't cause other processes to
		// fail. The publisher will re-create a new one.
		if err := n.ipfsNode.Repo.Datastore().Delete(ipnskey); err != nil {
			log.Errorf("Error deleting bad ipns record: %s", err)
		}
		return cid.Cid{}, err
	}
	return cid.Decode(strings.TrimPrefix(string(ourIpnsRecord.Value), "/ipfs/"))
}

// getFromDatastore looks in two places in the database for a record. First is
// under the /ipfs/<peerID> key which is sometimes used by the DHT. The value
// returned by this location is a serialized protobuf record. The second is
// under /ipfs/persistentcache/<peerID> which returns only the value (the path)
// inside the protobuf record.
func getFromDatastore(tx database.Tx, p peer.ID) (path.Path, error) {
	var entry models.CachedIPNSEntry
	if err := tx.Read().Where("peer_id = ?", p.String()).First(&entry).Error; err != nil {
		return nil, err
	}
	return path.New("/ipfs/" + entry.CID), nil
}

func putToDatastoreCache(tx database.Tx, p peer.ID, pth path.Path) error {
	entry := &models.CachedIPNSEntry{
		PeerID: p.String(),
		CID:    strings.TrimPrefix(pth.String(), "/ipfs/"),
	}
	return tx.Save(entry)
}

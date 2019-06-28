package core

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/models"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	nameopts "github.com/ipfs/interface-go-ipfs-core/options/namesys"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"strings"
	"time"
)

const (
	catTimeout     = time.Second * 30
	resolveTimeout = time.Second * 120
)

// cat fetches a file from IPFS given a path.
func (n *OpenBazaarNode) cat(pth path.Path) ([]byte, error) {
	catDone := make(chan struct{})
	defer func() {
		catDone <- struct{}{}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), catTimeout)

	api, err := coreapi.NewCoreAPI(n.ipfsNode)
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

	nd, err := api.Unixfs().Get(ctx, pth)
	if err != nil {
		return nil, err
	}

	r, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("incorrect type from Unixfs().Get()")
	}

	return ioutil.ReadAll(r)
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
func (n *OpenBazaarNode) resolve(p peer.ID, usecache bool) (path.Path, error) {
	if usecache {
		var pth path.Path
		err := n.repo.DB().View(func(tx *gorm.DB) error {
			var err error
			pth, err = getFromDatastore(tx, p)
			return err
		})
		if err == nil {
			// Update the cache in background
			go func() {
				pth, err := n.resolveOnce(p, resolveTimeout, n.ipnsQuorum)
				if err != nil {
					return
				}
				if n.ipfsNode.Identity != p {
					err = n.repo.DB().Update(func(tx *gorm.DB) error {
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
	pth, err := n.resolveOnce(p, resolveTimeout, n.ipnsQuorum)
	if err != nil {
		// Resolving fail. See if we have it in the db.
		var pth path.Path
		err := n.repo.DB().View(func(tx *gorm.DB) error {
			var err error
			pth, err = getFromDatastore(tx, p)
			return err
		})
		if err != nil {
			return nil, err
		}
		return pth, nil
	}
	// Resolving succeeded. Update the cache.
	if n.ipfsNode.Identity != p {
		err = n.repo.DB().Update(func(tx *gorm.DB) error {
			return putToDatastoreCache(tx, p, pth)
		})
		if err != nil {
			log.Error("Error putting IPNS record to datastore: %s", err.Error())
		}
	}
	return pth, nil
}

func (n *OpenBazaarNode) resolveOnce(p peer.ID, timeout time.Duration, quorum uint) (path.Path, error) {
	resolveDone := make(chan struct{})
	defer func() {
		resolveDone <- struct{}{}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

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

// getFromDatastore looks in two places in the database for a record. First is
// under the /ipfs/<peerID> key which is sometimes used by the DHT. The value
// returned by this location is a serialized protobuf record. The second is
// under /ipfs/persistentcache/<peerID> which returns only the value (the path)
// inside the protobuf record.
func getFromDatastore(tx *gorm.DB, p peer.ID) (path.Path, error) {
	var entry models.CachedIPNSEntry
	if err := tx.Where("peer_id = ?", p.String()).First(&entry).Error; err != nil {
		return nil, err
	}
	return path.New(string("/ipfs/" + entry.CID)), nil
}

func putToDatastoreCache(tx *gorm.DB, p peer.ID, pth path.Path) error {
	entry := &models.CachedIPNSEntry{
		PeerID: p.String(),
		CID:    strings.TrimPrefix(pth.String(), "/ipfs/"),
	}
	return tx.Save(entry).Error
}

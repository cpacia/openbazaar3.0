package core

import (
	"context"
	"errors"
	ds "github.com/ipfs/go-datastore"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/namesys"
	nameopts "github.com/ipfs/interface-go-ipfs-core/options/namesys"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"time"
)

const (
	catTimeout     = time.Second * 30
	resolveTimeout = time.Second * 120
)

// cat fetches the given path from IPFS.
func (n *OpenBazaarNode) cat(pth path.Path) ([]byte, error) {
	catDone := make(chan struct{})
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

// Resolve an IPNS record. This is a multi-step process.
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
		pth, err := getFromDatastore(n.ipfsNode.Repo.Datastore(), p)
		if err == nil {
			// Update the cache in background
			go func() {
				pth, err := n.resolveOnce(p, resolveTimeout, n.ipnsQuorum)
				if err != nil {
					return
				}
				if n.ipfsNode.Identity != p {
					if err := putToDatastore(n.ipfsNode.Repo.Datastore(), p, pth); err != nil {
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
		pth, err := getFromDatastore(n.ipfsNode.Repo.Datastore(), p)
		if err != nil {
			return nil, err
		}
		return pth, nil
	}
	// Resolving succeeded. Update the cache.
	if n.ipfsNode.Identity != p {
		if err := putToDatastore(n.ipfsNode.Repo.Datastore(), p, pth); err != nil {
			log.Error("Error putting IPNS record to datastore: %s", err.Error())
		}
	}
	return pth, nil
}

func (n *OpenBazaarNode) resolveOnce(p peer.ID, timeout time.Duration, quorum uint) (path.Path, error) {
	resolveDone := make(chan struct{})
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

func getFromDatastore(datastore ds.Datastore, p peer.ID) (path.Path, error) {
	// resolve to what we may already have in the datastore
	data, err := datastore.Get(namesys.IpnsDsKey(p))
	if err != nil {
		if err == ds.ErrNotFound {
			return nil, namesys.ErrResolveFailed
		}
		return nil, err
	}
	return path.New(string(data)), nil
}

func putToDatastore(datastore ds.Datastore, p peer.ID, pth path.Path) error {
	return datastore.Put(namesys.IpnsDsKey(p), []byte(pth.String()))
}

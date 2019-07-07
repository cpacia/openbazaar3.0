package core

import (
	"context"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/golang/protobuf/ptypes"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	fpath "github.com/ipfs/go-path"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"os"
	"os/signal"
)

// OpenBazaarNode holds all the components that make up a network node
// on the OpenBazaar network. It also exposes an exported API which can
// be used to control the node.
type OpenBazaarNode struct {

	// ipfsNode is the IPFS instance that powers this node.
	ipfsNode *core.IpfsNode

	// repo holds the database and public data directory.
	repo *repo.Repo

	// masterPrivKey represents an secp256k1 (HD) private key that
	// is advertised by the node in its profile and in listings to
	// be used when building escrow transactions.
	masterPrivKey *hdkeychain.ExtendedKey

	// ipnsQuorum is the size of the IPNS quorum to use. Smaller quorums
	// resolve faster but run the risk of getting back older records.
	ipnsQuorum uint

	// messenger is the primary object used to send messages to other peers.
	// It ensures reliable delivery by persisting messages and retrying them.
	// Generally you should always send messages using this and not the
	// NetworkService as the later will only attempt to send direct messages
	// and will not retry.
	messenger *net.Messenger

	// networkService manages the sending and receiving of messages
	// on the OpenBazaar protocol.
	networkService *net.NetworkService

	// banManager holds a list of peers that have been banned by this node.
	banManager *net.BanManager

	// eventBus allows a subscriber to receive event notifications from the node.
	eventBus events.Bus

	// followerTracker tries to maintain connections to a minimum number of our
	// followers so that we can use them to push data for redundancy.
	followerTracker *FollowerTracker

	// multiwallet is a map of cyptocurrency wallets.
	//multiwallet multiwallet.MultiWallet

	// testnet is whether the this node is configured to use the test network.
	testnet bool

	// shutdown is closed when the node is stopped. Any listening
	// goroutines can use this to terminate.
	shutdown chan struct{}
}

// Start gets the node up and running and listens for a signal interrupt.
func (n *OpenBazaarNode) Start() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Info("OpenBazaar shutting down...")
			n.Stop()
			os.Exit(1)
		}
	}()
	go n.messenger.Start()
	go n.followerTracker.Start()
	go n.syncMessages()
}

// Stop cleanly shutsdown the OpenBazaarNode and signals to any
// listening goroutines that it's time to stop.
func (n *OpenBazaarNode) Stop() {
	close(n.shutdown)
	n.ipfsNode.Close()
	n.repo.Close()
	n.networkService.Close()
	n.messenger.Stop()
}

// UsingTestnet returns whether or not this node is running on
// the test network.
func (n *OpenBazaarNode) UsingTestnet() bool {
	return n.testnet
}

// DestroyNode shutsdown the node and deletes the entire data directory.
// This should only be used during testing as destroying a live node will
// result in data loss.
func (n *OpenBazaarNode) DestroyNode() {
	n.Stop()
	n.repo.DestroyRepo()
}

// IPFSNode returns the underlying IPFS node instance.
func (n *OpenBazaarNode) IPFSNode() *core.IpfsNode {
	return n.ipfsNode
}

// Identity returns the peer ID for this node.
func (n *OpenBazaarNode) Identity() peer.ID {
	return n.ipfsNode.Identity
}

// Publish will publish the current public data directory to IPNS.
// It will interrupt the publish if a shutdown happens during.
func (n *OpenBazaarNode) Publish(done chan<- struct{}) {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			select {
			case <-ctx.Done():
			case <-n.shutdown:
				cancel()
			}
			if done != nil {
				close(done)
			}
		}()

		api, err := coreapi.NewCoreAPI(n.ipfsNode)
		if err != nil {
			log.Errorf("Error building core API: %s", err.Error())
			return
		}

		currentRoot, err := n.ipnsRecordValue()

		// First uppin old root hash
		if err == nil {
			rp, err := api.ResolvePath(context.Background(), ipath.IpfsPath(currentRoot))
			if err != nil {
				log.Errorf("Error resolving path: %s", err.Error())
				return
			}

			if err := api.Pin().Rm(context.Background(), rp, options.Pin.RmRecursive(true)); err != nil {
				log.Errorf("Error unpinning root: %s", err.Error())
				return
			}
		}

		// Add the directory to IPFS
		stat, err := os.Lstat(n.repo.DB().PublicDataPath())
		if err != nil {
			log.Errorf("Error calling Lstat: %s", err.Error())
			return
		}

		f, err := files.NewSerialFile(n.repo.DB().PublicDataPath(), false, stat)
		if err != nil {
			log.Errorf("Error serializing file: %s", err.Error())
			return
		}

		opts := []options.UnixfsAddOption{
			options.Unixfs.Pin(true),
		}
		pth, err := api.Unixfs().Add(context.Background(), files.ToDir(f), opts...)
		if err != nil {
			log.Errorf("Error adding root: %s", err.Error())
			return
		}

		// Publish
		if err := n.ipfsNode.Namesys.Publish(ctx, n.ipfsNode.PrivateKey, fpath.FromString(pth.Root().String())); err != nil {
			log.Errorf("Error namesys publish: %s", err.Error())
			return
		}

		// Send the new graph to our connected followers.
		graph, err := n.fetchGraph()
		if err != nil {
			log.Errorf("Error fetching graph: %s", err.Error())
			return
		}

		storeMsg := &pb.StoreMessage{}
		for _, cid := range graph {
			storeMsg.Cids = append(storeMsg.Cids, cid.Bytes())
		}

		any, err := ptypes.MarshalAny(storeMsg)
		if err != nil {
			log.Errorf("Error marshalling store message: %s", err.Error())
			return
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_STORE
		msg.Payload = any
		for _, peer := range n.followerTracker.ConnectedFollowers() {
			go n.networkService.SendMessage(context.Background(), peer, msg)
		}
	}()
}

// SubscribeEvent returns a subscription to the provided event. The event argument
// may be an interface slice.
func (n *OpenBazaarNode) SubscribeEvent(event interface{}) (events.Subscription, error) {
	return n.eventBus.Subscribe(event)
}

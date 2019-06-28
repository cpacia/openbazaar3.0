package core

import (
	"context"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/ipfs/go-ipfs/core"
	"github.com/jinzhu/gorm"
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
		if err := n.repo.PublicData().Publish(ctx, n.ipfsNode); err != nil {
			log.Errorf("Publish error: %s", err.Error())
		}
	}()
}

func (n *OpenBazaarNode) handleAckMessage(from peer.ID, message *pb.Message) error {
	return n.repo.DBUpdate(func(tx *gorm.DB) error {
		return n.messenger.ProcessACK(tx, message)
	})
}

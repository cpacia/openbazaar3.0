package core

import (
	"context"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	repo "github.com/cpacia/openbazaar3.0/repo"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/bootstrap"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	"github.com/ipfs/go-ipfs/namesys"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

// MockNode builds a mock node with a temp data directory,
// in-memory database, mock IPFS node, and mock network
// service.
func MockNode() (*OpenBazaarNode, error) {
	r, err := repo.MockRepo()
	if err != nil {
		return nil, err
	}

	ipfsNode, err := coremock.NewMockNode()
	if err != nil {
		return nil, err
	}

	banManager := net.NewBanManager(nil)
	service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

	messenger := net.NewMessenger(service, r.DB())

	var dbSeed models.Key
	err = r.DB().View(func(tx database.Tx) error {
		return tx.DB().Where("name = ?", "seed").First(&dbSeed).Error
	})

	masterPrivKey, err := hdkeychain.NewMaster(dbSeed.Value, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	bus := events.NewBus()
	tracker := NewFollowerTracker(r, bus, ipfsNode.PeerHost.Network())

	node := &OpenBazaarNode{
		ipfsNode:        ipfsNode,
		repo:            r,
		networkService:  service,
		messenger:       messenger,
		eventBus:        bus,
		banManager:      banManager,
		ipnsQuorum:      1,
		shutdown:        make(chan struct{}),
		masterPrivKey:   masterPrivKey,
		followerTracker: tracker,
	}

	node.registerHandlers()
	node.listenNetworkEvents()
	return node, nil
}

// MockNet represents a network of connected mock nodes.
type Mocknet struct {
	nodes   []*OpenBazaarNode
	ipfsNet mocknet.Mocknet
}

// NewMocknet returns a new MockNet without the
// nodes connected to each other.
func NewMocknet(numNodes int) (*Mocknet, error) {
	ctx := context.Background()

	// create network
	mn := mocknet.New(ctx)

	var nodes []*OpenBazaarNode
	for i := 0; i < numNodes; i++ {
		ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
			Online: true,
			Host:   coremock.MockHostOption(mn),
		})
		if err != nil {
			return nil, err
		}

		ipfsNode.Namesys = namesys.NewNameSystem(ipfsNode.Routing, ipfsNode.Repo.Datastore(), 0)

		r, err := repo.MockRepo()
		if err != nil {
			return nil, err
		}

		banManager := net.NewBanManager(nil)
		service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

		messenger := net.NewMessenger(service, r.DB())

		var dbSeed models.Key
		err = r.DB().View(func(tx database.Tx) error {
			return tx.DB().Where("name = ?", "seed").First(&dbSeed).Error
		})

		masterPrivKey, err := hdkeychain.NewMaster(dbSeed.Value, &chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}

		bus := events.NewBus()
		tracker := NewFollowerTracker(r, bus, ipfsNode.PeerHost.Network())

		node := &OpenBazaarNode{
			ipfsNode:        ipfsNode,
			repo:            r,
			networkService:  service,
			messenger:       messenger,
			eventBus:        bus,
			banManager:      banManager,
			ipnsQuorum:      1,
			shutdown:        make(chan struct{}),
			masterPrivKey:   masterPrivKey,
			followerTracker: tracker,
		}

		node.registerHandlers()
		node.listenNetworkEvents()

		nodes = append(nodes, node)
	}

	if err := mn.LinkAll(); err != nil {
		return nil, err
	}

	bsinf := bootstrap.BootstrapConfigWithPeers(
		[]peerstore.PeerInfo{
			nodes[0].ipfsNode.Peerstore.PeerInfo(nodes[0].Identity()),
		},
	)

	for _, n := range nodes[1:] {
		if err := n.ipfsNode.Bootstrap(bsinf); err != nil {
			return nil, err
		}
	}

	return &Mocknet{nodes, mn}, nil
}

// Nodes returns the OpenBazaar nodes in this network.
func (mn *Mocknet) Nodes() []*OpenBazaarNode {
	return mn.nodes
}

// Peers returns the peer IDs of the nodes in the network.
func (mn *Mocknet) Peers() []peer.ID {
	return mn.ipfsNet.Peers()
}

// StartAll starts all nodes in the network.
func (mn *Mocknet) StartAll() {
	for _, n := range mn.nodes {
		n.Start()
	}
}

// TearDown shutsdown the network and destroys the data directories.
func (mn *Mocknet) TearDown() error {
	for _, n := range mn.nodes {
		n.Stop()
		if err := n.repo.DestroyRepo(); err != nil {
			return err
		}
	}
	return nil
}

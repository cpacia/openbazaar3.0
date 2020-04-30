package core

import (
	"context"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/orders"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/bootstrap"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"path"
)

// MockNode builds a mock node with a temp data directory,
// in-memory database, mock IPFS node, and mock network
// service.
func MockNode() (*OpenBazaarNode, error) {
	r, err := repo.MockRepo()
	if err != nil {
		return nil, err
	}

	ipfsRepo, err := fsrepo.Open(path.Join(r.DataDir(), "ipfs"))
	if err != nil {
		return nil, err
	}

	ipfsConfig, err := ipfsRepo.Config()
	if err != nil {
		return nil, err
	}

	ipfsConfig.Bootstrap = nil

	var dbIdentityKey models.Key
	err = r.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	ipfsConfig.Identity, err = repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	mn := mocknet.New(ctx)

	ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
		Online: true,
		Repo:   ipfsRepo,
		Host:   coremock.MockHostOption(mn),
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	})
	if err != nil {
		return nil, err
	}

	banManager := net.NewBanManager(nil)
	service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

	// Load the keys from the db
	var (
		dbEscrowKey models.Key
		dbRatingKey models.Key
	)
	err = r.DB().View(func(tx database.Tx) error {
		if err := tx.Read().Where("name = ?", "escrow").First(&dbEscrowKey).Error; err != nil {
			return err
		}
		return tx.Read().Where("name = ?", "ratings").First(&dbRatingKey).Error
	})

	escrowKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbEscrowKey.Value)
	ratingKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbRatingKey.Value)

	bus := events.NewBus()
	tracker := NewFollowerTracker(r, bus, ipfsNode.PeerHost)

	w := wallet.NewMockWallet()
	w.SetEventBus(bus)

	mw := multiwallet.Multiwallet{
		iwallet.CtMock: w,
	}

	erp, err := wallet.NewMockExchangeRates()
	if err != nil {
		return nil, err
	}

	node := &OpenBazaarNode{
		ipfsNode:             ipfsNode,
		repo:                 r,
		networkService:       service,
		eventBus:             bus,
		banManager:           banManager,
		ipnsQuorum:           1,
		shutdown:             make(chan struct{}),
		escrowMasterKey:      escrowKey,
		ratingMasterKey:      ratingKey,
		multiwallet:          mw,
		followerTracker:      tracker,
		exchangeRates:        erp,
		initialBootstrapChan: make(chan struct{}),
		publishChan:          make(chan pubCloser),
	}

	node.messenger, err = net.NewMessenger(&net.MessengerConfig{
		Privkey: ipfsNode.PrivateKey,
		Service: service,
		DB:      r.DB(),
		Context: ipfsNode.Context(),
	})
	if err != nil {
		return nil, err
	}
	node.orderProcessor = orders.NewOrderProcessor(&orders.Config{
		Identity:             ipfsNode.Identity,
		IdentityPrivateKey:   ipfsNode.PrivateKey,
		Db:                   r.DB(),
		Multiwallet:          mw,
		Messenger:            node.messenger,
		EscrowPrivateKey:     escrowKey,
		ExchangeRateProvider: erp,
		EventBus:             bus,
		CalcCIDFunc:          node.cid,
	})

	node.registerHandlers()
	node.listenNetworkEvents()
	node.publishHandler()
	close(node.initialBootstrapChan)
	return node, nil
}

// MockNet represents a network of connected mock nodes.
type Mocknet struct {
	nodes   []*OpenBazaarNode
	ipfsNet mocknet.Mocknet
	wn      *wallet.MockWalletNetwork
}

// NewMocknet returns a new MockNet without the
// nodes connected to each other.
func NewMocknet(numNodes int) (*Mocknet, error) {
	ctx := context.Background()

	// create network
	mn := mocknet.New(ctx)

	wn := wallet.NewMockWalletNetwork(numNodes)

	//bootstrap.DefaultBootstrapConfig.MinPeerThreshold = 1

	var nodes []*OpenBazaarNode
	for i := 0; i < numNodes; i++ {
		r, err := repo.MockRepo()
		if err != nil {
			return nil, err
		}

		ipfsRepo, err := fsrepo.Open(path.Join(r.DataDir(), "ipfs"))
		if err != nil {
			return nil, err
		}

		ipfsConfig, err := ipfsRepo.Config()
		if err != nil {
			return nil, err
		}

		ipfsConfig.Bootstrap = nil

		var dbIdentityKey models.Key
		err = r.DB().View(func(tx database.Tx) error {
			return tx.Read().Where("name = ?", "identity").First(&dbIdentityKey).Error
		})

		ipfsConfig.Identity, err = repo.IdentityFromKey(dbIdentityKey.Value)
		if err != nil {
			return nil, err
		}

		ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
			Online: true,
			Repo:   ipfsRepo,
			Host:   coremock.MockHostOption(mn),
			ExtraOpts: map[string]bool{
				"pubsub": true,
			},
			Routing: constructMockRouting,
		})
		if err != nil {
			return nil, err
		}

		ipfsNode.Namesys = namesys.NewNameSystem(ipfsNode.Routing, ipfsNode.Repo.Datastore(), 0)

		banManager := net.NewBanManager(nil)
		service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

		// Load the keys from the db
		var (
			dbEscrowKey models.Key
			dbRatingKey models.Key
		)
		err = r.DB().View(func(tx database.Tx) error {
			if err := tx.Read().Where("name = ?", "escrow").First(&dbEscrowKey).Error; err != nil {
				return err
			}
			return tx.Read().Where("name = ?", "ratings").First(&dbRatingKey).Error
		})

		escrowKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbEscrowKey.Value)
		ratingKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbRatingKey.Value)

		bus := events.NewBus()
		tracker := NewFollowerTracker(r, bus, ipfsNode.PeerHost)

		w := wn.Wallets()[i]
		w.SetEventBus(bus)

		mw := multiwallet.Multiwallet{
			iwallet.CtMock: w,
		}

		erp, err := wallet.NewMockExchangeRates()
		if err != nil {
			return nil, err
		}

		node := &OpenBazaarNode{
			ipfsNode:             ipfsNode,
			repo:                 r,
			networkService:       service,
			eventBus:             bus,
			banManager:           banManager,
			ipnsQuorum:           1,
			shutdown:             make(chan struct{}),
			escrowMasterKey:      escrowKey,
			ratingMasterKey:      ratingKey,
			multiwallet:          mw,
			followerTracker:      tracker,
			exchangeRates:        erp,
			initialBootstrapChan: make(chan struct{}),
			publishChan:          make(chan pubCloser),
		}

		node.messenger, err = net.NewMessenger(&net.MessengerConfig{
			Privkey: ipfsNode.PrivateKey,
			Service: service,
			DB:      r.DB(),
			Context: ipfsNode.Context(),
		})
		if err != nil {
			return nil, err
		}
		node.orderProcessor = orders.NewOrderProcessor(&orders.Config{
			Identity:             ipfsNode.Identity,
			IdentityPrivateKey:   ipfsNode.PrivateKey,
			Db:                   r.DB(),
			Messenger:            node.messenger,
			Multiwallet:          mw,
			EscrowPrivateKey:     escrowKey,
			ExchangeRateProvider: erp,
			EventBus:             bus,
			CalcCIDFunc:          node.cid,
		})

		node.registerHandlers()
		node.listenNetworkEvents()
		node.publishHandler()
		close(node.initialBootstrapChan)

		nodes = append(nodes, node)
	}

	if err := mn.LinkAll(); err != nil {
		return nil, err
	}

	bsinf := bootstrap.BootstrapConfigWithPeers(
		[]peer.AddrInfo{
			nodes[0].ipfsNode.Peerstore.PeerInfo(nodes[0].Identity()),
		},
	)

	for _, n := range nodes[1:] {
		if err := n.ipfsNode.Bootstrap(bsinf); err != nil {
			return nil, err
		}
	}

	return &Mocknet{nodes, mn, wn}, nil
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

func (mn *Mocknet) StartWalletNetwork() {
	mn.wn.Start()
}

// WalletNetwork returns the mock wallet network.
func (mn *Mocknet) WalletNetwork() *wallet.MockWalletNetwork {
	return mn.wn
}

// TearDown shutsdown the network and destroys the data directories.
func (mn *Mocknet) TearDown() error {
	for _, n := range mn.nodes {
		if n == nil {
			continue
		}
		n.Stop(true)
		if err := n.repo.DestroyRepo(); err != nil {
			return err
		}
	}
	return nil
}

func constructMockRouting(ctx context.Context, host host.Host, dstore datastore.Batching, validator record.Validator) (routing.Routing, error) {
	return dht.New(
		ctx, host,
		dht.Concurrency(10),
		dht.Mode(dht.ModeServer),
		dht.Datastore(dstore),
		dht.Validator(validator),
		dht.ProtocolPrefix(ProtocolDHT),
		dht.MaxRecordAge(maxRecordAge),
	)
}

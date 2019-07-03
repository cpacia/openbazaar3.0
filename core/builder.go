package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/repo"
	bitswap "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/jinzhu/gorm"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing"
	"github.com/op/go-logging"
)

var (
	log         = logging.MustGetLogger("CORE")
	ProtocolDHT protocol.ID
)

// NewNode constructs and returns an IpfsNode using the given cfg.
func NewNode(ctx context.Context, cfg *repo.Config) (*OpenBazaarNode, error) {
	obRepo, err := repo.NewRepo(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	// Load the IPFS Repo
	ipfsRepo, err := fsrepo.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	ipfsConfig, err := ipfsRepo.Config()
	if err != nil {
		return nil, err
	}

	// If bootstrap addresses were provided in the config, override the IPFS defaults.
	if len(cfg.BoostrapAddrs) > 0 {
		ipfsConfig.Bootstrap = cfg.BoostrapAddrs
	}

	// If swarm addresses were provided in the config, override the IPFS defaults.
	if len(cfg.SwarmAddrs) > 0 {
		ipfsConfig.Addresses.Swarm = cfg.SwarmAddrs
	}

	// If a gateway address was provided in the config, override the IPFS default.
	if cfg.GatewayAddr != "" {
		ipfsConfig.Addresses.Gateway = config.Strings{cfg.GatewayAddr}
	}

	// Load our identity key from the db and set it in the config.
	var dbIdentityKey models.Key
	err = obRepo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	ipfsConfig.Identity, err = repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		return nil, err
	}

	// Update the protocol IDs for Bitswap and the Kad-DHT. This is used to segregate the
	// network from mainline IPFS.
	updateIPFSGlobalProtocolVars(cfg.Testnet)
	if !cfg.Testnet {
		ProtocolDHT = net.ProtocolKademliaMainnetTwo
	} else {
		ProtocolDHT = net.ProtocolKademliaTestnetTwo
	}

	// New IPFS build config
	ncfg := &core.BuildCfg{
		Repo:   ipfsRepo,
		Online: true,
		ExtraOpts: map[string]bool{
			"mplex":  true,
			"ipnsps": true,
		},
		Routing: constructRouting,
	}

	// Construct IPFS node.
	ipfsNode, err := core.NewNode(ctx, ncfg)
	if err != nil {
		return nil, err
	}

	// Load the seed from the db so we can build the masterPrivKey
	var dbSeed models.Key
	err = obRepo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("name = ?", "seed").First(&dbSeed).Error
	})

	masterPrivKey, err := hdkeychain.NewMaster(dbSeed.Value, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	bus := events.NewBus()
	bm := net.NewBanManager(nil) // TODO: load ids from db
	service := net.NewNetworkService(ipfsNode.PeerHost, bm, cfg.Testnet)
	messenger := net.NewMessenger(service, obRepo.DB())
	tracker := NewFollowerTracker(obRepo, bus, ipfsNode.PeerHost.Network())

	// Construct our OpenBazaar node.repo object
	obNode := &OpenBazaarNode{
		ipfsNode:        ipfsNode,
		repo:            obRepo,
		masterPrivKey:   masterPrivKey,
		ipnsQuorum:      cfg.IPNSQuorum,
		messenger:       messenger,
		networkService:  service,
		banManager:      bm,
		eventBus:        bus,
		followerTracker: tracker,
		shutdown:        make(chan struct{}),
	}

	obNode.registerHandlers()
	obNode.listenNetworkEvents()

	return obNode, nil
}

// constructRouting behaves exactly like the default constructRouting function in the IPFS package
// with the loan exception of setting the dhtopts.Protocols to use our custom protocol ID. By using
// a different ID we ensure that we segregate the OpenBazaar DHT from the main IPFS DHT.
func constructRouting(ctx context.Context, host host.Host, dstore datastore.Batching, validator record.Validator) (routing.IpfsRouting, error) {
	return dht.New(
		ctx, host,
		dhtopts.Datastore(dstore),
		dhtopts.Validator(validator),
		dhtopts.Protocols(
			ProtocolDHT,
		),
	)
}

func updateIPFSGlobalProtocolVars(testnetEnable bool) {
	if testnetEnable {
		bitswap.ProtocolBitswap = net.ProtocolBitswapMainnetTwo
		bitswap.ProtocolBitswapOne = net.ProtocolBitswapMainnetTwoDotOne
		bitswap.ProtocolBitswapNoVers = net.ProtocolBitswapMainnetNoVers
	} else {
		bitswap.ProtocolBitswap = net.ProtocolBitswapTestnetTwo
		bitswap.ProtocolBitswapOne = net.ProtocolBitswapTestnetTwoDotOne
		bitswap.ProtocolBitswapNoVers = net.ProtocolBitswapTestnetNoVers
	}
}

func (n *OpenBazaarNode) registerHandlers() {
	n.networkService.RegisterHandler(pb.Message_CHAT, n.handleChatMessage)
	n.networkService.RegisterHandler(pb.Message_ACK, n.handleAckMessage)
	n.networkService.RegisterHandler(pb.Message_FOLLOW, n.handleFollowMessage)
	n.networkService.RegisterHandler(pb.Message_UNFOLLOW, n.handleUnFollowMessage)
}

func (n *OpenBazaarNode) listenNetworkEvents() {
	connected := func(_ inet.Network, conn inet.Conn) {
		n.eventBus.Emit(&events.PeerConnected{Peer: conn.RemotePeer()})
	}
	disConnected := func(_ inet.Network, conn inet.Conn) {
		n.eventBus.Emit(&events.PeerDisconnected{Peer: conn.RemotePeer()})
	}

	notifier := &inet.NotifyBundle{
		ConnectedF:    connected,
		DisconnectedF: disConnected,
	}

	n.ipfsNode.PeerHost.Network().Notify(notifier)
}

// newMessageWithID returns a new *pb.Message with a random
// message ID.
func newMessageWithID() *pb.Message {
	messageID := make([]byte, 20)
	rand.Read(messageID)
	return &pb.Message{
		MessageID: hex.EncodeToString(messageID),
	}
}

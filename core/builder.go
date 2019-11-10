package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/api"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	obnet "github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/cpacia/openbazaar3.0/wallet"
	bitswap "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/op/go-logging"
	"net"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
)

var (
	log             = logging.MustGetLogger("CORE")
	stdoutLogFormat = logging.MustStringFormatter(`%{color:reset}%{color}%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`)
	fileLogFormat   = logging.MustStringFormatter(`%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`)
	ProtocolDHT     protocol.ID
)

// NewNode constructs and returns an IpfsNode using the given cfg.
func NewNode(ctx context.Context, cfg *repo.Config) (*OpenBazaarNode, error) {
	obRepo, err := repo.NewRepo(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	// Load the IPFS Repo
	ipfsRepo, err := fsrepo.Open(path.Join(cfg.DataDir, "ipfs"))
	if err != nil {
		return nil, err
	}

	ipfsConfig, err := ipfsRepo.Config()
	if err != nil {
		return nil, err
	}

	// Disable MDNS
	ipfsConfig.Swarm.DisableNatPortMap = cfg.DisableNATPortMap

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

	// Profiling
	if cfg.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			log.Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			log.Errorf("%v", http.ListenAndServe(listenAddr, nil))
		}()
	}

	// Write cpu profile if requested.
	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			log.Errorf("Unable to create cpu profile: %v", err)
			return nil, err
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	// Load our identity key from the db and set it in the config.
	var dbIdentityKey models.Key
	err = obRepo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	ipfsConfig.Identity, err = repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		return nil, err
	}

	// Update the protocol IDs for Bitswap and the Kad-DHT. This is used to segregate the
	// network from mainline IPFS.
	updateIPFSGlobalProtocolVars(cfg.Testnet)
	if !cfg.Testnet {
		ProtocolDHT = obnet.ProtocolKademliaMainnetTwo
	} else {
		ProtocolDHT = obnet.ProtocolKademliaTestnetTwo
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

	// Load the keys from the db
	var (
		dbEscrowKey models.Key
		dbRatingKey models.Key
	)
	err = obRepo.DB().View(func(tx database.Tx) error {
		if err := tx.Read().Where("name = ?", "escrow").First(&dbEscrowKey).Error; err != nil {
			return err
		}
		return tx.Read().Where("name = ?", "ratings").First(&dbRatingKey).Error
	})

	escrowKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbEscrowKey.Value)
	ratingKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbRatingKey.Value)

	bus := events.NewBus()
	bm := obnet.NewBanManager(nil) // TODO: load ids from db
	service := obnet.NewNetworkService(ipfsNode.PeerHost, bm, cfg.Testnet)
	messenger := obnet.NewMessenger(service, obRepo.DB())
	tracker := NewFollowerTracker(obRepo, bus, ipfsNode.PeerHost.Network())

	mw := wallet.Multiwallet{} // TODO: wire this up.

	erp := wallet.NewExchangeRateProvider(nil, cfg.ExchangeRateProviders) // TODO: wire up proxy

	op := orders.NewOrderProcessor(&orders.Config{
		Identity:             ipfsNode.Identity,
		Db:                   obRepo.DB(),
		Messenger:            messenger,
		Multiwallet:          mw,
		ExchangeRateProvider: erp,
		EventBus:             bus,
	})

	// Construct our OpenBazaar node.repo object
	obNode := &OpenBazaarNode{
		ipfsNode:        ipfsNode,
		repo:            obRepo,
		escrowMasterKey: escrowKey,
		ratingMasterKey: ratingKey,
		ipnsQuorum:      cfg.IPNSQuorum,
		messenger:       messenger,
		networkService:  service,
		banManager:      bm,
		eventBus:        bus,
		followerTracker: tracker,
		multiwallet:     mw,
		exchangeRates:   erp,
		orderProcessor:  op,
		testnet:         cfg.Testnet,
		shutdown:        make(chan struct{}),
	}

	obNode.gateway, err = obNode.newHTTPGateway(cfg)
	if err != nil {
		return nil, err
	}

	obNode.registerHandlers()
	obNode.listenNetworkEvents()

	return obNode, nil
}

func NewIPFSOnlyNode(ctx context.Context, dataDir string, testnet bool) (*core.IpfsNode, error) {
	// Load the IPFS Repo
	ipfsRepo, err := fsrepo.Open(path.Join(dataDir, "ipfs"))
	if err != nil {
		return nil, err
	}
	ncfg := &core.BuildCfg{
		Repo:   ipfsRepo,
		Online: true,
		ExtraOpts: map[string]bool{
			"mplex":  true,
			"ipnsps": true,
		},
		Routing: constructRouting,
	}

	updateIPFSGlobalProtocolVars(testnet)
	if !testnet {
		ProtocolDHT = obnet.ProtocolKademliaMainnetTwo
	} else {
		ProtocolDHT = obnet.ProtocolKademliaTestnetTwo
	}

	// Construct IPFS node.
	return core.NewNode(ctx, ncfg)
}

type dummyListener struct {
	addr net.Addr
}

func (d *dummyListener) Addr() net.Addr {
	return d.addr
}
func (d *dummyListener) Accept() (net.Conn, error) {
	conn, _ := net.FileConn(nil)
	return conn, nil
}
func (d *dummyListener) Close() error {
	return nil
}

func (n *OpenBazaarNode) newHTTPGateway(cfg *repo.Config) (*api.Gateway, error) {
	// Get API configuration
	ipfsConf, err := n.ipfsNode.Repo.Config()
	if err != nil {
		return nil, err
	}

	// Create a network listener
	gatewayMaddr, err := ma.NewMultiaddr(ipfsConf.Addresses.Gateway[0])
	if err != nil {
		return nil, fmt.Errorf("newHTTPGateway: invalid gateway address: %q (err: %s)", ipfsConf.Addresses.Gateway, err)
	}
	var gwLis manet.Listener
	if cfg.UseSSL {
		netAddr, err := manet.ToNetAddr(gatewayMaddr)
		if err != nil {
			return nil, err
		}
		gwLis, err = manet.WrapNetListener(&dummyListener{netAddr})
		if err != nil {
			return nil, err
		}
	} else {
		gwLis, err = manet.Listen(gatewayMaddr)
		if err != nil {
			return nil, fmt.Errorf("newHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
		}
	}

	// We might have listened to /tcp/0 - let's see what we are listing on
	gatewayMaddr = gwLis.Multiaddr()
	log.Infof("Gateway/API server listening on %s\n", gatewayMaddr)

	// Setup an options slice
	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.VersionOption(),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(ipfsConf.Gateway.Writable, "/ipfs", "/ipns"),
	}

	if len(ipfsConf.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", ipfsConf.Gateway.RootRedirect))
	}

	allowedIPs := make(map[string]bool)
	for _, ip := range cfg.APIAllowedIPs {
		allowedIPs[ip] = true
	}

	config := &api.GatewayConfig{
		Listener:   manet.NetListener(gwLis),
		NoCors:     cfg.APINoCors,
		UseSSL:     cfg.UseSSL,
		SSLCert:    cfg.SSLCertFile,
		SSLKey:     cfg.SSLKeyFile,
		Username:   cfg.APIUsername,
		Password:   cfg.APIPassword,
		Cookie:     cfg.APICookie,
		PublicOnly: cfg.APIPublicGateway,
		AllowedIPs: allowedIPs,
	}

	return api.NewGateway(n, config, opts...)
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
		bitswap.ProtocolBitswap = obnet.ProtocolBitswapMainnetTwo
		bitswap.ProtocolBitswapOne = obnet.ProtocolBitswapMainnetTwoDotOne
		bitswap.ProtocolBitswapNoVers = obnet.ProtocolBitswapMainnetNoVers
	} else {
		bitswap.ProtocolBitswap = obnet.ProtocolBitswapTestnetTwo
		bitswap.ProtocolBitswapOne = obnet.ProtocolBitswapTestnetTwoDotOne
		bitswap.ProtocolBitswapNoVers = obnet.ProtocolBitswapTestnetNoVers
	}
}

func (n *OpenBazaarNode) registerHandlers() {
	n.networkService.RegisterHandler(pb.Message_CHAT, n.handleChatMessage)
	n.networkService.RegisterHandler(pb.Message_ACK, n.handleAckMessage)
	n.networkService.RegisterHandler(pb.Message_FOLLOW, n.handleFollowMessage)
	n.networkService.RegisterHandler(pb.Message_UNFOLLOW, n.handleUnFollowMessage)
	n.networkService.RegisterHandler(pb.Message_STORE, n.handleStoreMessage)
	n.networkService.RegisterHandler(pb.Message_ORDER, n.handleOrderMessage)
	n.networkService.RegisterHandler(pb.Message_ADDRESS_REQUEST, n.handleAddressRequest)
	n.networkService.RegisterHandler(pb.Message_ADDRESS_RESPONSE, n.handleAddressResponse)
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

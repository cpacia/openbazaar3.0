package core

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/hdkeychain"
	storeandforward "github.com/cpacia/go-store-and-forward"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/api"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	obnet "github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/notifications"
	"github.com/cpacia/openbazaar3.0/orders"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/cpacia/openbazaar3.0/wallet"
	"github.com/cpacia/proxyclient"
	iwallet "github.com/cpacia/wallet-interface"
	bitswap "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/jinzhu/gorm"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p-routing"
	lcfg "github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/op/go-logging"
	"net"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"time"
)

var (
	log             = logging.MustGetLogger("CORE")
	stdoutLogFormat = logging.MustStringFormatter(`%{color:reset}%{color}%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`)
	fileLogFormat   = logging.MustStringFormatter(`%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`)
	ProtocolDHT     protocol.ID
)

// NewNode constructs and returns an OpenBazaarNode using the given cfg.
func NewNode(ctx context.Context, cfg *repo.Config) (*OpenBazaarNode, error) {
	obRepo, err := repo.NewRepo(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	repo.SetupLogging(cfg.LogDir, cfg.LogLevel)

	if err := obRepo.WriteUserAgent(cfg.UserAgentComment); err != nil {
		return nil, err
	}

	var (
		transportOpt    libp2p.Option
		onionID         string
		shutdownTorFunc func() error
	)
	if cfg.Tor || cfg.DualStack {
		log.Notice("Starting embedded Tor client")

		var torKey models.Key
		err = obRepo.DB().View(func(tx database.Tx) error {
			return tx.Read().Where("name = ?", "tor").First(&torKey).Error
		})
		if err != nil {
			return nil, err
		}

		key := ed25519.NewKeyFromSeed(torKey.Value)

		onion, dialer, transport, closeTor, err := obnet.SetupTor(ctx, key, cfg.DataDir, cfg.DualStack)
		if err != nil {
			return nil, err
		}
		onionID = onion
		transportOpt = transport
		shutdownTorFunc = closeTor

		if cfg.Tor {
			proxyclient.SetProxy(dialer)
		}
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
	if cfg.Tor {
		ipfsConfig.Addresses.Swarm = []string{fmt.Sprintf("/onion3/%s:9003", onionID)}
	} else if cfg.DualStack {
		ipfsConfig.Addresses.Swarm = append(ipfsConfig.Addresses.Swarm, fmt.Sprintf("/onion3/%s:9003", onionID))
	}

	// If a gateway address was provided in the config, override the IPFS default.
	if cfg.GatewayAddr != "" {
		ipfsConfig.Addresses.Gateway = config.Strings{cfg.GatewayAddr}
	}

	if cfg.Tor {
		ipfsConfig.Swarm.DisableNatPortMap = true
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

	constructPeerHost := func(ctx context.Context, id peer.ID, ps peerstore.Peerstore, options ...libp2p.Option) (host.Host, error) {
		pkey := ps.PrivKey(id)
		if pkey == nil {
			return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
		}
		options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)

		config := &lcfg.Config{}
		if err := config.Apply(options...); err != nil {
			return nil, err
		}

		if cfg.Tor {
			config.Transports = []lcfg.TptC{}
			if err := transportOpt(config); err != nil {
				return nil, err
			}
		} else if cfg.DualStack {
			if err := transportOpt(config); err != nil {
				return nil, err
			}
		}
		return config.NewNode(ctx)
	}

	// New IPFS build config
	ncfg := &core.BuildCfg{
		Repo:      ipfsRepo,
		Online:    true,
		Permanent: true,
		ExtraOpts: map[string]bool{
			"mplex":  true,
			"ipnsps": true,
			"pubsub": true,
		},
		Routing: constructRouting,
		Host:    constructPeerHost,
	}

	// Construct IPFS node.
	ipfsNode, err := core.NewNode(ctx, ncfg)
	if err != nil {
		return nil, err
	}

	// Store and forward client and server
	snfServers := make([]peer.ID, 0, len(cfg.StoreAndForwardServers))
	for _, serverStr := range cfg.StoreAndForwardServers {
		server, err := peer.IDB58Decode(serverStr)
		if err != nil {
			return nil, err
		}
		snfServers = append(snfServers, server)
	}

	snfProtocol := obnet.ProtocolStoreAndForwardMainnet
	if cfg.Testnet {
		snfProtocol = obnet.ProtocolStoreAndForwardTestnet
	}

	if cfg.EnableSNFServer {
		snfReplicationPeers := make([]peer.ID, 0, len(cfg.SNFServerPeers))
		for _, serverStr := range cfg.SNFServerPeers {
			server, err := peer.IDB58Decode(serverStr)
			if err != nil {
				return nil, err
			}
			snfReplicationPeers = append(snfReplicationPeers, server)
		}
		serverOpts := []storeandforward.Option{
			storeandforward.Protocols(protocol.ID(snfProtocol)),
			storeandforward.ReplicationPeers(snfReplicationPeers...),
			storeandforward.Datastore(ipfsNode.Repo.Datastore()),
		}
		_, err := storeandforward.NewServer(ipfsNode.Context(), ipfsNode.PeerHost, serverOpts...)
		if err != nil {
			return nil, err
		}
	}

	if cfg.IPFSOnly {
		return &OpenBazaarNode{
			repo:            obRepo,
			ipfsNode:        ipfsNode,
			ipfsOnlyMode:    true,
			shutdownTorFunc: shutdownTorFunc,
			eventBus:        events.NewBus(),
			shutdown:        make(chan struct{}),
		}, nil
	}

	// Load the keys from the db
	var (
		dbEscrowKey models.Key
		dbRatingKey models.Key
		prefs       models.UserPreferences
	)
	err = obRepo.DB().View(func(tx database.Tx) error {
		if err := tx.Read().First(&prefs).Error; err != nil {
			return err
		}
		if err := tx.Read().Where("name = ?", "escrow").First(&dbEscrowKey).Error; err != nil {
			return err
		}
		return tx.Read().Where("name = ?", "ratings").First(&dbRatingKey).Error
	})

	escrowKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbEscrowKey.Value)
	ratingKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), dbRatingKey.Value)

	bus := events.NewBus()

	blocked, err := prefs.BlockedNodes()
	if err != nil {
		return nil, err
	}
	bm := obnet.NewBanManager(blocked)
	service := obnet.NewNetworkService(ipfsNode.PeerHost, bm, cfg.Testnet)
	tracker := NewFollowerTracker(obRepo, bus, ipfsNode.PeerHost)

	enabledWallets := make([]iwallet.CoinType, len(cfg.EnabledWallets))
	for i, ew := range cfg.EnabledWallets {
		enabledWallets[i] = iwallet.CoinType(strings.ToUpper(ew))
	}

	opts := []multiwallet.Option{
		multiwallet.DataDir(cfg.DataDir),
		multiwallet.LogDir(cfg.LogDir),
		multiwallet.Wallets(enabledWallets),
		multiwallet.LogLevel(repo.LogLevelMap[strings.ToLower(cfg.LogLevel)]),
	}
	mw, err := multiwallet.NewMultiwallet(opts...)
	if err != nil {
		return nil, err
	}

	for ct, wallet := range mw {
		// Create wallet if not exists. This will fail if the bip44 key has been deleted
		// from the db, however we are not yet deleting keys or the mnemonic for encryption
		// purposes.
		if !wallet.WalletExists() {
			var bip44ModelKey models.Key
			err = obRepo.DB().View(func(tx database.Tx) error {
				return tx.Read().Where("name = ?", "bip44").First(&bip44ModelKey).Error
			})
			if gorm.IsRecordNotFoundError(err) {
				return nil, fmt.Errorf("can not initialize wallet %s: seed does not exist in database", ct.CurrencyCode())
			} else if err != nil {
				return nil, err
			}

			bip44Key, err := hdkeychain.NewKeyFromString(string(bip44ModelKey.Value))
			if err != nil {
				return nil, err
			}

			def, err := models.CurrencyDefinitions.Lookup(ct.CurrencyCode())
			if err != nil {
				return nil, err
			}

			coinTypeKey, err := bip44Key.Child(hdkeychain.HardenedKeyStart + uint32(def.Bip44Code))
			if err != nil {
				return nil, err
			}

			accountKey, err := coinTypeKey.Child(hdkeychain.HardenedKeyStart + 0)
			if err != nil {
				return nil, err
			}

			if err := wallet.CreateWallet(*accountKey, nil, time.Now()); err != nil {
				return nil, err
			}
		}
	}

	erp := wallet.NewExchangeRateProvider(cfg.ExchangeRateProviders)

	for _, server := range cfg.StoreAndForwardServers {
		_, err := peer.IDB58Decode(server)
		if err != nil {
			return nil, errors.New("invalid store and forward peer ID in config")
		}
	}

	// Construct our OpenBazaar node.repo object
	obNode := &OpenBazaarNode{
		ipfsNode:               ipfsNode,
		repo:                   obRepo,
		escrowMasterKey:        escrowKey,
		ratingMasterKey:        ratingKey,
		ipnsQuorum:             cfg.IPNSQuorum,
		networkService:         service,
		banManager:             bm,
		eventBus:               bus,
		followerTracker:        tracker,
		multiwallet:            mw,
		exchangeRates:          erp,
		testnet:                cfg.Testnet,
		storeAndForwardServers: cfg.StoreAndForwardServers,
		shutdownTorFunc:        shutdownTorFunc,
		publishChan:            make(chan pubCloser),
		initialBootstrapChan:   make(chan struct{}),
		shutdown:               make(chan struct{}),
	}

	obNode.gateway, err = obNode.newHTTPGateway(cfg)
	if err != nil {
		return nil, err
	}

	obNode.notifier = notifications.NewNotifier(bus, obRepo.DB(), obNode.gateway.NotifyWebsockets)
	obNode.messenger, err = obnet.NewMessenger(&obnet.MessengerConfig{
		Service:        service,
		SNFServers:     snfServers,
		Privkey:        ipfsNode.PrivateKey,
		Context:        ipfsNode.Context(),
		DB:             obRepo.DB(),
		Testnet:        cfg.Testnet,
		GetProfileFunc: obNode.GetProfile,
	})
	if err != nil {
		return nil, err
	}

	obNode.orderProcessor = orders.NewOrderProcessor(&orders.Config{
		Identity:             ipfsNode.Identity,
		IdentityPrivateKey:   ipfsNode.PrivateKey,
		Db:                   obRepo.DB(),
		Multiwallet:          mw,
		Messenger:            obNode.messenger,
		EscrowPrivateKey:     escrowKey,
		ExchangeRateProvider: erp,
		EventBus:             bus,
		CalcCIDFunc:          obNode.cid,
	})

	obNode.registerHandlers()
	obNode.listenNetworkEvents()

	return obNode, nil
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
	n.networkService.RegisterHandler(pb.Message_PING, n.handlePingMessage)
	n.networkService.RegisterHandler(pb.Message_PONG, n.handlePongMessage)
}

func (n *OpenBazaarNode) listenNetworkEvents() {
	serverMap := make(map[string]bool)
	for _, server := range n.storeAndForwardServers {
		serverMap[server] = true
	}

	connected := func(_ inet.Network, conn inet.Conn) {
		if serverMap[conn.RemotePeer().Pretty()] {
			log.Debugf("Established connection to store and forward server %s", conn.RemotePeer().Pretty())
		}
		n.eventBus.Emit(&events.PeerConnected{Peer: conn.RemotePeer()})
	}
	disConnected := func(_ inet.Network, conn inet.Conn) {
		if serverMap[conn.RemotePeer().Pretty()] {
			log.Debugf("Disconnected from store and forward server %s", conn.RemotePeer().Pretty())
		}
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

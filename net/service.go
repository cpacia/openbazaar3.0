package net

import (
	"context"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	ctxio "github.com/jbenet/go-context/io"
	host "github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	msgio "github.com/libp2p/go-msgio"
	"github.com/op/go-logging"
	"io"
	"sync"
)

var log = logging.MustGetLogger("NET")

type NetworkService struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	host host.Host

	messageSenders map[peer.ID]*messageSender

	msMtx sync.RWMutex

	handlers   map[pb.Message_MessageType]func(peerID peer.ID, msg *pb.Message) error
	handlerMtx sync.RWMutex

	banManager *BanManager

	protocolID protocol.ID
}

func NewNetworkService(host host.Host, banManager *BanManager, useTestnet bool) *NetworkService {
	ctx, cancel := context.WithCancel(context.Background())
	protocolID := ProtocolAppMainnetTwo
	if useTestnet {
		protocolID = ProtocolAppTestnetTwo
	}
	ns := &NetworkService{
		ctx:            ctx,
		ctxCancel:      cancel,
		host:           host,
		messageSenders: make(map[peer.ID]*messageSender),
		msMtx:          sync.RWMutex{},
		handlers:       make(map[pb.Message_MessageType]func(peerID peer.ID, message *pb.Message) error),
		handlerMtx:     sync.RWMutex{},
		banManager:     banManager,
		protocolID:     protocol.ID(protocolID),
	}

	disConnected := func(_ inet.Network, conn inet.Conn) {
		ns.msMtx.Lock()
		defer ns.msMtx.Unlock()
		delete(ns.messageSenders, conn.RemotePeer())
	}
	notifier := &inet.NotifyBundle{
		DisconnectedF: disConnected,
	}
	host.Network().Notify(notifier)
	host.SetStreamHandler(ns.protocolID, ns.HandleNewStream)
	return ns
}

func (ns *NetworkService) Close() {
	ns.ctxCancel()
}

func (ns *NetworkService) RegisterHandler(messageType pb.Message_MessageType, handler func(peerID peer.ID, message *pb.Message) error) {
	ns.handlerMtx.Lock()
	defer ns.handlerMtx.Unlock()
	ns.handlers[messageType] = handler
}

// HandleNewStream receives new incoming streams from other peers.
// A stream is not a connection. You may already have an open connection
// with this peer over which you have been using other protocols. A stream
// is an abstraction which allows you to multiplex multiple streams of data
// over the same connection. Each stream does not technically need to be
// a different protocol. You could, for example, have multiple streams open
// to the same peer using the OpenBazaarProtocol. This would allow for each
// stream operating in parallel with each other *as if* each one were a
// different connection.
func (ns *NetworkService) HandleNewStream(s inet.Stream) {
	go ns.handleNewMessage(s)
}

func (ns *NetworkService) handleNewMessage(s inet.Stream) {
	defer s.Close()
	contextReader := ctxio.NewReader(ns.ctx, s)
	reader := msgio.NewVarintReaderSize(contextReader, inet.MessageSizeMax)
	remotePeer := s.Conn().RemotePeer()

	if ns.banManager.IsBanned(remotePeer) {
		log.Debugf("Received new stream request from banned peer %s. Closing.", remotePeer)
		return
	}

	for {
		select {
		case <-ns.ctx.Done():
			return
		default:
		}

		pmes := new(pb.Message)
		msgBytes, err := reader.ReadMsg()
		if err != nil {
			reader.ReleaseMsg(msgBytes)
			s.Reset()
			if err == io.EOF {
				log.Debugf("Peer %s closed stream", remotePeer)
			}
			return
		}
		if err := proto.Unmarshal(msgBytes, pmes); err != nil {
			reader.ReleaseMsg(msgBytes)
			s.Reset()
			return
		}
		reader.ReleaseMsg(msgBytes)
		// Check again
		if ns.banManager.IsBanned(remotePeer) {
			log.Debugf("Received message from banned peer %s. Closing.", remotePeer)
			return
		}

		ns.handlerMtx.RLock()
		handler, ok := ns.handlers[pmes.MessageType]
		if !ok {
			log.Warningf("Received message type %s with unregistered handler", pmes.MessageType.String())
			ns.handlerMtx.RUnlock()
			continue
		}
		ns.handlerMtx.RUnlock()
		if err := handler(remotePeer, pmes); err != nil {
			log.Errorf("Error processing %s message from %s: %s", pmes.MessageType.String(), remotePeer, err)
		}
	}
}

func (ns *NetworkService) SendMessage(ctx context.Context, peerID peer.ID, message *pb.Message) error {
	ms, err := ns.messageSenderForPeer(ctx, peerID)
	if err != nil {
		return err
	}
	return ms.sendMessage(ctx, message)
}

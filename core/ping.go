package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/net/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

const maxPongDelay = time.Second * 10

// PingNode sends a PING message to the provided peer. If we are able to successfully
// connect and receive an PONG message back we return nil. If we don't receive a
// PONG message back an error is returned.
func (n *OpenBazaarNode) PingNode(ctx context.Context, peer peer.ID) error {
	sub, err := n.eventBus.Subscribe(&events.PongReceived{})
	if err != nil {
		return err
	}
	defer sub.Close()

	m := newMessageWithID()
	m.MessageType = pb.Message_PING
	if err := n.networkService.SendMessage(ctx, peer, m); err != nil {
		return err
	}

	select {
	case <-sub.Out():
		return nil
	case <-ctx.Done():
	case <-time.After(maxPongDelay):
	}
	return ErrPeerUnreachable
}

func (n *OpenBazaarNode) handlePingMessage(from peer.ID, message *pb.Message) error {
	n.eventBus.Emit(&events.PingReceived{
		Peer: from,
	})
	m := newMessageWithID()
	m.MessageType = pb.Message_PONG
	return n.networkService.SendMessage(context.Background(), from, m)
}

func (n *OpenBazaarNode) handlePongMessage(from peer.ID, message *pb.Message) error {
	n.eventBus.Emit(&events.PongReceived{
		Peer: from,
	})
	return nil
}

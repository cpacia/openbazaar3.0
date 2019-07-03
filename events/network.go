package events

import (
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-peer"
)

// PeerConnected is an event that gets pushed to the bus
// whenever a new peer connects.
type PeerConnected struct {
	Peer peer.ID
}

// PeerDisconnected is an event that gets pushed to the bus
// whenever a peer disconnects.
type PeerDisconnected struct {
	Peer peer.ID
}

// MessageACK is an event that gets pushed to the bus
// whenever a message ACK is received.
type MessageACK struct {
	MessageID string
}

// MessageStore is an event that gets pushed to the bus
// whenever a STORE message is received.
type MessageStore struct {
	Peer peer.ID
	Cids []cid.Cid
}

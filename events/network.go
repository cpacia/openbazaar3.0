package events

import peer "github.com/libp2p/go-libp2p-peer"

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

package models

import (
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// CachedIPNSEntry holds a cached IPNS recrod. This is stored in our
// database.
type CachedIPNSEntry struct {
	PeerID string `gorm:"primaryKey"`
	CID    string
}

// GetPeerID returns the peer.ID object for this entry.
func (e *CachedIPNSEntry) GetPeerID() (peer.ID, error) {
	return peer.Decode(e.PeerID)
}

// GetCID returns the CID object which is the value for
// this entry.
func (e *CachedIPNSEntry) GetCID() (cid.Cid, error) {
	return cid.Decode(e.CID)
}

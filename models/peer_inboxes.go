package models

import (
	"encoding/json"
	peer "github.com/libp2p/go-libp2p-peer"
)

type PeerInboxes struct {
	PeerID     string `gorm:"primary_key"`
	InboxPeers json.RawMessage
}

// PutInboxes save the peer IDs into the model. This will override
// the existing peer IDs.
func (p *PeerInboxes) PutInboxes(peerIDs []string) error {
	out, err := json.Marshal(peerIDs)
	if err != nil {
		return err
	}
	p.InboxPeers = out
	return nil
}

// Inboxes returns the peer IDs saved in the model.
func (p *PeerInboxes) Inboxes() ([]peer.ID, error) {
	if p.InboxPeers == nil {
		return nil, nil
	}
	var inboxPeers []string
	if err := json.Unmarshal(p.InboxPeers, &inboxPeers); err != nil {
		return nil, err
	}
	pids := make([]peer.ID, 0, len(inboxPeers))
	for _, peerStr := range inboxPeers {
		pid, err := peer.IDB58Decode(peerStr)
		if err != nil {
			return nil, err
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

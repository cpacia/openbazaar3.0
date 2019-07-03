package core

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/repo"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"os"
	"time"
)

const (
	// FollowerConnections is the number of followers the tracker
	// attempts to maintain connections to.
	FollowerConnections = 10

	// TrackerInterval is how frequently the tracker runs in
	// attempt to connect to more peers.
	TrackerInterval = time.Minute * 2
)

// FollowerTracker is an object which tracks the uptime and responsiveness
// of all of the nodes that follow us. It will use that information to
// attempt to maintain connections to a handful of known good followers
// so that we can use them to pin data for us.
type FollowerTracker struct {
	connected map[peer.ID]bool
	repo      *repo.Repo
	bus       events.Bus
	net       inet.Network
}

// NewFollowerTracker returns a new FollowerTracker which has
// not yet been started.
func NewFollowerTracker(repo *repo.Repo, bus events.Bus, net inet.Network) *FollowerTracker {
	return &FollowerTracker{
		connected: make(map[peer.ID]bool),
		repo:      repo,
		bus:       bus,
		net:       net,
	}
}

// Start runs the follower tracker. It will start by loading allow followers
// from the db and seeing which if any we are connected to. It will then
// try to maintain active connects to a handful of followers but will
// slowly attempt connections so as to not slam the network with traffic.
//
// It also will receive notifications of connected and disconnected peers
// and act on that information accordingly.
//
// This must be run in a separate goroutine.
func (t *FollowerTracker) Start() {
	followers, err := t.repo.PublicData().GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		log.Error("Error loading followers: %s", err)
	}

peerLoop:
	for _, peer := range t.net.Peers() {
		for _, follower := range followers {
			if peer.Pretty() == follower {
				t.connected[peer] = true
				continue peerLoop
			}
		}
	}
	go t.tryConnectFollowers()

	ticker := time.NewTicker(TrackerInterval)
	for range ticker.C {
		go t.tryConnectFollowers()
	}

}

func (t *FollowerTracker) listenEvents() {
	/*connectedSub, err := t.bus.Subscribe(&events.PeerConnected{})
	if err != nil {
		log.Error("Error subscribing to PeerConnected event: %s", err)
	}
	disonnectedSub, err := t.bus.Subscribe(&events.PeerDisconnected{})
	if err != nil {
		log.Error("Error subscribing to PeerDisconnected event: %s", err)
	}
	for {
	}*/

}

func (t *FollowerTracker) tryConnectFollowers() {

}
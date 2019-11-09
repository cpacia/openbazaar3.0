package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/jinzhu/gorm"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	"os"
	"sync"
	"time"
)

const (
	// FollowerConnections is the number of followers the tracker
	// attempts to maintain connections to.
	FollowerConnections = 10

	// TrackerInterval is how frequently the tracker runs in
	// attempt to connect to more peers.
	TrackerInterval = time.Second * 30

	// TrackAttemptsPerInterval is the number of connections attempted
	// per interval.
	TrackerAttemptsPerInterval = 3
)

// FollowerTracker is an object which tracks the uptime and responsiveness
// of all of the nodes that follow us. It will use that information to
// attempt to maintain connections to a handful of known good followers
// so that we can use them to pin data for us.
type FollowerTracker struct {
	connected  map[peer.ID]time.Time
	triedPeers map[peer.ID]time.Time
	followers  map[peer.ID]bool
	peerCh     chan peer.ID
	mtx        sync.RWMutex
	repo       *repo.Repo
	bus        events.Bus
	net        inet.Network
	shutdown   chan struct{}
}

// NewFollowerTracker returns a new FollowerTracker which has
// not yet been started.
func NewFollowerTracker(repo *repo.Repo, bus events.Bus, net inet.Network) *FollowerTracker {
	return &FollowerTracker{
		connected:  make(map[peer.ID]time.Time),
		triedPeers: make(map[peer.ID]time.Time),
		peerCh:     make(chan peer.ID),
		followers:  make(map[peer.ID]bool),
		shutdown:   make(chan struct{}),
		mtx:        sync.RWMutex{},
		repo:       repo,
		bus:        bus,
		net:        net,
	}
}

// Start runs the follower tracker. It will start by loading allow followers
// from the db and seeing which if any we are connected to. It will then
// try to maintain active connects to a handful of followers but will
// slowly attempt connections so as to not slam the network with traffic.
//
// It also will receive notifications of connected and disconnected peers
// and act on that information accordingly.
func (t *FollowerTracker) Start() {
	var (
		followers models.Followers
		err       error
	)
	err = t.repo.DB().View(func(tx database.Tx) error {
		followers, err = tx.GetFollowers()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		log.Error("Error loading followers: %s", err)
	}

	for _, follower := range followers {
		pid, err := peer.IDB58Decode(follower)
		if err != nil {
			log.Errorf("Error unmarshalling peerID: %s", err)
			continue
		}
		t.followers[pid] = true
	}

	for _, peer := range t.net.Peers() {
		if _, ok := t.followers[peer]; ok {
			t.connected[peer] = time.Now()
		}
	}
	go t.listenEvents()
}

func (t *FollowerTracker) Close() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	for pid, connectedTime := range t.connected {
		connectedDuration := time.Since(connectedTime)

		err := t.repo.DB().Update(func(tx database.Tx) error {
			var stat models.FollowerStat
			if err := tx.Read().Where("peer_id = ?", pid.Pretty()).First(&stat).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
				return err
			}
			stat.PeerID = pid.Pretty()
			stat.LastConnection = time.Now()
			stat.ConnectedDuration += connectedDuration
			return tx.Save(&stat)
		})
		if err != nil {
			log.Errorf("Follower Tracker Close Error: %s", err)
		}
		delete(t.connected, pid)
	}

	close(t.shutdown)
}

// ConnectedFollowers returns a slice of peers that can be
// used to push data to.
func (t *FollowerTracker) ConnectedFollowers() []peer.ID {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	peers := make([]peer.ID, 0, FollowerConnections)
	for connected := range t.connected {
		peers = append(peers, connected)
		if len(peers) >= FollowerConnections {
			break
		}
	}
	return peers
}

func (t *FollowerTracker) listenEvents() {
	connectedSub, err := t.bus.Subscribe(&events.PeerConnected{})
	if err != nil {
		log.Error("Error subscribing to PeerConnected event: %s", err)
	}
	disonnectedSub, err := t.bus.Subscribe(&events.PeerDisconnected{})
	if err != nil {
		log.Error("Error subscribing to PeerDisconnected event: %s", err)
	}
	followerSub, err := t.bus.Subscribe(&events.FollowNotification{})
	if err != nil {
		log.Error("Error subscribing to FollowNotification event: %s", err)
	}
	unfollowerSub, err := t.bus.Subscribe(&events.UnfollowNotification{})
	if err != nil {
		log.Error("Error subscribing to UnfollowNotification event: %s", err)
	}
	ticker := time.NewTicker(TrackerInterval)
	t.bus.Emit(&events.TrackerStarted{})
	for {
		select {
		case <-ticker.C:
			go t.tryConnectFollowers()
		case event := <-connectedSub.Out():
			notif, ok := event.(*events.PeerConnected)
			if !ok {
				log.Error("Follower tracker type assertion failed on PeerConnected")
				continue
			}
			t.mtx.Lock()
			if _, ok := t.followers[notif.Peer]; ok {
				t.connected[notif.Peer] = time.Now()
			}
			t.mtx.Unlock()
			t.bus.Emit(&events.TrackerPeerConnected{Peer: notif.Peer})
		case event := <-disonnectedSub.Out():
			notif, ok := event.(*events.PeerDisconnected)
			if !ok {
				log.Error("Follower tracker type assertion failed on PeerDisconnected")
				continue
			}
			t.mtx.Lock()
			if _, ok := t.followers[notif.Peer]; ok {
				connectedTime, ok := t.connected[notif.Peer]
				if !ok {
					t.mtx.Unlock()
					continue
				}
				connectedDuration := time.Since(connectedTime)

				err := t.repo.DB().Update(func(tx database.Tx) error {
					var stat models.FollowerStat
					if err := tx.Read().Where("peer_id = ?", notif.Peer.Pretty()).First(&stat).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
						return err
					}
					stat.PeerID = notif.Peer.Pretty()
					stat.LastConnection = time.Now()
					stat.ConnectedDuration += connectedDuration
					return tx.Save(&stat)
				})
				if err != nil {
					log.Error(err)
				}

				delete(t.connected, notif.Peer)
			}
			t.mtx.Unlock()
			t.bus.Emit(&events.TrackerPeerDisconnected{Peer: notif.Peer})
		case event := <-followerSub.Out():
			notif, ok := event.(*events.FollowNotification)
			if !ok {
				log.Error("Follower tracker type assertion failed on FollowNotification")
				continue
			}
			pid, err := peer.IDB58Decode(notif.PeerID)
			if err != nil {
				log.Errorf("Error unmarshalling peerID: %s", err)
				continue
			}
			t.mtx.Lock()
			t.followers[pid] = true
			t.mtx.Unlock()
			t.bus.Emit(&events.TrackerFollow{Peer: pid})

		case event := <-unfollowerSub.Out():
			notif, ok := event.(*events.UnfollowNotification)
			if !ok {
				log.Error("Follower tracker type assertion failed on UnfollowNotification")
				continue
			}
			pid, err := peer.IDB58Decode(notif.PeerID)
			if err != nil {
				log.Error(err)
				continue
			}
			t.mtx.Lock()
			delete(t.followers, pid)
			t.mtx.Unlock()
			t.bus.Emit(&events.TrackerUnfollow{Peer: pid})

		case pid := <-t.peerCh:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			go func() {
				defer cancel()
				t.net.DialPeer(ctx, pid)
			}()

		case <-t.shutdown:
			return
		}
	}

}

func (t *FollowerTracker) tryConnectFollowers() {
	// Loop through the network peers to see if we don't have
	// any marked as connected.
	t.mtx.Lock()
	for _, peer := range t.net.Peers() {
		if _, ok := t.followers[peer]; ok {
			if _, aok := t.connected[peer]; !aok {
				t.connected[peer] = time.Now()
			}
		}
	}
	t.mtx.Unlock()
	// If we already have enough connections then exit.
	t.mtx.RLock()
	if len(t.connected) >= FollowerConnections {
		t.mtx.RUnlock()
		return
	}
	t.mtx.RUnlock()

	// First try connecting to known good followers.
	var stats []models.FollowerStat
	err := t.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Order("connected_duration asc").Order("last_connection asc").Find(&stats).Error
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		log.Errorf("Error loading followers from db %s", err)
	}

	attempts := 0
	for _, stat := range stats {
		if attempts < TrackerAttemptsPerInterval {
			if t.tryFollower(stat.PeerID) {
				attempts++
			}
		}
	}

	// Then move on to the rest of the followers.
	var followers models.Followers
	err = t.repo.DB().View(func(tx database.Tx) error {
		followers, err = tx.GetFollowers()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		log.Error(err)
	}

	for _, follower := range followers {
		if attempts < TrackerAttemptsPerInterval {
			if t.tryFollower(follower) {
				attempts++
			}
		}
	}
}

func (t *FollowerTracker) tryFollower(p string) bool {
	pid, err := peer.IDB58Decode(p)
	if err != nil {
		log.Error()
		return false
	}
	t.mtx.RLock()
	if _, ok := t.connected[pid]; ok {
		t.mtx.RUnlock()
		return false
	}
	if lastAttempt, ok := t.triedPeers[pid]; ok {
		if lastAttempt.Before(time.Now().Add(time.Minute * 30)) {
			t.mtx.RUnlock()
			return false
		}
	}
	t.mtx.RUnlock()

	t.mtx.Lock()
	t.peerCh <- pid
	t.triedPeers[pid] = time.Now()
	t.mtx.Unlock()

	return true
}

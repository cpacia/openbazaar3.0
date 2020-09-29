package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"gorm.io/gorm"
	"os"
)

// FollowNode updates the local repo with the new following peer,
// updates the stats in the profile and sends the follow message to the
// remote peer. A publish is not done here if you want the data to show
// on the network immediately you must manually trigger a publish.
func (n *OpenBazaarNode) FollowNode(peerID peer.ID, done chan<- struct{}) error {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		following, err := tx.GetFollowing()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		for _, peer := range following {
			if peer == peerID.Pretty() {
				return fmt.Errorf("%w: already following peer", coreiface.ErrBadRequest)
			}
		}

		var seq models.FollowSequence
		if err := tx.Read().Where("peer_id = ?", peerID.Pretty()).First(&seq).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		seq.Num++
		seq.PeerID = peerID.Pretty()
		if err := tx.Save(&seq); err != nil {
			return err
		}

		following = append(following, peerID.Pretty())

		if err := tx.SetFollowing(following); err != nil {
			return err
		}

		if err := n.updateAndSaveProfile(tx); err != nil {
			return err
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_FOLLOW
		msg.Sequence = uint32(seq.Num)

		log.Debugf("Sending FOLLOW message to %s. MessageID: %s", peerID, msg.MessageID)
		if err := n.messenger.ReliablySendMessage(tx, peerID, msg, done); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
		return err
	}
	return nil
}

// UnfollowNode deletes the following peer from the local repo,
// updates the stats in the profile and sends the unfollow message to the
// remote peer. A publish is not done here if you want the data to show
// on the network immediately you must manually trigger a publish.
func (n *OpenBazaarNode) UnfollowNode(peerID peer.ID, done chan<- struct{}) error {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		following, err := tx.GetFollowing()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		exists := false
		for i, pid := range following {
			if pid == peerID.Pretty() {
				exists = true
				following = append(following[:i], following[i+1:]...)
				break
			}
		}
		if !exists {
			return fmt.Errorf("%w: not following peer", coreiface.ErrBadRequest)
		}

		var seq models.FollowSequence
		if err := tx.Read().Where("peer_id = ?", peerID.Pretty()).First(&seq).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		seq.PeerID = peerID.Pretty()
		seq.Num++
		if err := tx.Save(&seq); err != nil {
			return err
		}

		if err := tx.SetFollowing(following); err != nil {
			return err
		}

		if err := n.updateAndSaveProfile(tx); err != nil {
			return err
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_UNFOLLOW
		msg.Sequence = uint32(seq.Num)

		log.Debugf("Sending UNFOLLOW message to %s. MessageID: %s", peerID, msg.MessageID)
		if err := n.messenger.ReliablySendMessage(tx, peerID, msg, done); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
	}
	return nil
}

// GetMyFollowers returns the followers list for this node.
func (n *OpenBazaarNode) GetMyFollowers() (models.Followers, error) {
	var (
		followers models.Followers
		err       error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		followers, err = tx.GetFollowers()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return followers, nil
}

// GetMyFollowing returns the following list for this node.
func (n *OpenBazaarNode) GetMyFollowing() (models.Following, error) {
	var (
		following models.Following
		err       error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		following, err = tx.GetFollowing()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return following, nil
}

// GetFollowers returns the followers of the node with the given peer ID.
// If useCache is set it will return the followers from the local cache
// (if it has one) if followers file is not found on the network.
func (n *OpenBazaarNode) GetFollowers(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	followersBytes, err := n.cat(ctx, path.Join(pth, ffsqlite.FollowersFile))
	if err != nil {
		return nil, err
	}
	var followers models.Followers
	if err := json.Unmarshal(followersBytes, &followers); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	for _, f := range followers {
		if _, err := peer.Decode(f); err != nil {
			return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
		}
	}
	return followers, nil
}

// GetFollowing returns the following of the node with the given peer ID.
// If useCache is set it will return the following from the local cache
// (if it has one) if following file is not found on the network.
func (n *OpenBazaarNode) GetFollowing(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	followersBytes, err := n.cat(ctx, path.Join(pth, ffsqlite.FollowingFile))
	if err != nil {
		return nil, err
	}
	var following models.Following
	if err := json.Unmarshal(followersBytes, &following); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	for _, f := range following {
		if _, err := peer.Decode(f); err != nil {
			return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
		}
	}
	return following, nil
}

// handleFollowMessage handles incoming follow messages from the network.
func (n *OpenBazaarNode) handleFollowMessage(from peer.ID, message *pb.Message) error {
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	var ErrAlreadyFollowing = errors.New("peer already following us")

	err := n.repo.DB().Update(func(tx database.Tx) error {
		followers, err := tx.GetFollowers()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		for _, follower := range followers {
			if follower == from.Pretty() {
				return ErrAlreadyFollowing
			}
		}
		followers = append(followers, from.Pretty())

		return tx.SetFollowers(followers)
	})
	if err != nil && err != ErrAlreadyFollowing {
		return err
	} else if err == ErrAlreadyFollowing {
		log.Debugf("Received FOLLOW message from peer %s which already follows us", from)
		return nil
	}

	log.Infof("Received FOLLOW message from %s", from)
	n.eventBus.Emit(&events.Follow{
		PeerID: from.Pretty(),
	})
	return nil
}

// handleUnFollowMessage handles incoming unfollow messages from the network.
func (n *OpenBazaarNode) handleUnFollowMessage(from peer.ID, message *pb.Message) error {
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	log.Infof("Received UNFOLLOW message from %s", from)

	var ErrNotFollowing = errors.New("peer not following us")

	err := n.repo.DB().Update(func(tx database.Tx) error {
		followers, err := tx.GetFollowers()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		exists := false
		for i, pid := range followers {
			if pid == from.Pretty() {
				exists = true
				followers = append(followers[:i], followers[i+1:]...)
				break
			}
		}
		if !exists {
			return ErrNotFollowing
		}

		return tx.SetFollowers(followers)
	})
	if err != nil && err != ErrNotFollowing {
		return err
	} else if err == ErrNotFollowing {
		log.Debugf("Received UNFOLLOW message from peer %s that was not following us", from)
		return nil
	}

	n.eventBus.Emit(&events.Unfollow{
		PeerID: from.Pretty(),
	})
	return nil
}

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"os"
)

// FollowNode updates the local repo with the new following peer,
// updates the stats in the profile and sends the follow message to the
// remote peer. A publish is not done here if you want the data to show
// on the network immediately you must manually trigger a publish.
func (n *OpenBazaarNode) FollowNode(peerID peer.ID, done chan<- struct{}) error {
	return n.repo.DB().Update(func(tx *gorm.DB) error {
		following, err := n.repo.PublicData().GetFollowing()
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		for _, peer := range following {
			if peer == peerID.Pretty() {
				return errors.New("already following peer")
			}
		}

		var seq models.FollowSequence
		if err := tx.Where("peer_id = ?", peerID.Pretty()).First(&seq).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		seq.Num++
		seq.PeerID = peerID.Pretty()
		if err := tx.Save(&seq).Error; err != nil {
			return err
		}

		following = append(following, peerID.Pretty())

		if err := n.repo.PublicData().SetFollowing(following); err != nil {
			return err
		}

		if err := n.updateAndSaveProfile(); err != nil {
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
}

// UnfollowNode deletes the following peer from the local repo,
// updates the stats in the profile and sends the unfollow message to the
// remote peer. A publish is not done here if you want the data to show
// on the network immediately you must manually trigger a publish.
func (n *OpenBazaarNode) UnfollowNode(peerID peer.ID, done chan<- struct{}) error {
	return n.repo.DB().Update(func(tx *gorm.DB) error {
		following, err := n.repo.PublicData().GetFollowing()
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
			return errors.New("not following peer")
		}

		var seq models.FollowSequence
		if err := tx.Where("peer_id = ?", peerID.Pretty()).First(&seq).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		seq.PeerID = peerID.Pretty()
		seq.Num++
		if err := tx.Save(&seq).Error; err != nil {
			return err
		}

		if err := n.repo.PublicData().SetFollowing(following); err != nil {
			return err
		}

		if err := n.updateAndSaveProfile(); err != nil {
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
}

// GetMyFollowers returns the followers list for this node.
func (n *OpenBazaarNode) GetMyFollowers() (models.Followers, error) {
	return n.repo.PublicData().GetFollowers()
}

// GetMyFollowing returns the following list for this node.
func (n *OpenBazaarNode) GetMyFollowing() (models.Following, error) {
	return n.repo.PublicData().GetFollowing()
}

// GetFollowers returns the followers of the node with the given peer ID.
// If useCache is set it will return the followers from the local cache
// (if it has one) if followers file is not found on the network.
func (n *OpenBazaarNode) GetFollowers(peerID peer.ID, useCache bool) (models.Followers, error) {
	pth, err := n.resolve(peerID, useCache)
	if err != nil {
		return nil, err
	}
	followersBytes, err := n.cat(path.Join(pth, repo.FollowersFile))
	if err != nil {
		return nil, err
	}
	var followers models.Followers
	if err := json.Unmarshal(followersBytes, &followers); err != nil {
		return nil, err
	}
	for _, f := range followers {
		if _, err := peer.IDB58Decode(f); err != nil {
			return nil, err
		}
	}
	return followers, nil
}

// GetFollowing returns the following of the node with the given peer ID.
// If useCache is set it will return the following from the local cache
// (if it has one) if following file is not found on the network.
func (n *OpenBazaarNode) GetFollowing(peerID peer.ID, useCache bool) (models.Following, error) {
	pth, err := n.resolve(peerID, useCache)
	if err != nil {
		return nil, err
	}
	followersBytes, err := n.cat(path.Join(pth, repo.FollowingFile))
	if err != nil {
		return nil, err
	}
	var following models.Following
	if err := json.Unmarshal(followersBytes, &following); err != nil {
		return nil, err
	}
	for _, f := range following {
		if _, err := peer.IDB58Decode(f); err != nil {
			return nil, err
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

	log.Infof("Received FOLLOW message from %s", from)

	followers, err := n.repo.PublicData().GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, follower := range followers {
		if follower == from.Pretty() {
			log.Debugf("Received FOLLOW message from peer %s which already follows us", from)
			return nil
		}
	}
	followers = append(followers, from.Pretty())

	if err := n.repo.PublicData().SetFollowers(followers); err != nil {
		return err
	}
	n.eventBus.Emit(&events.FollowNotification{
		PeerID: from.Pretty(),
		ID:     message.MessageID,
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

	followers, err := n.repo.PublicData().GetFollowers()
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
		return fmt.Errorf("received UNFOLLOW message from peer %s that was not following us", from)
	}

	if err := n.repo.PublicData().SetFollowers(followers); err != nil {
		return err
	}
	n.eventBus.Emit(&events.UnfollowNotification{
		PeerID: from.Pretty(),
		ID:     message.MessageID,
	})
	return nil
}

package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
)

// sendAckMessage saves the incoming message ID in the database so we can
// check for duplicate messages later. Then it sends the ACK message to
// the remote peer.
func (n *OpenBazaarNode) sendAckMessage(messageID string, to peer.ID) {
	err := n.repo.DB().Update(func(tx *gorm.DB) error {
		return tx.Save(&models.IncomingMessage{ID: messageID}).Error
	})
	if err != nil {
		log.Errorf("Error saving incoming message ID to database: %s", err)
	}
	n.messenger.SendACK(messageID, to)
}

// handleAckMessage is the handler for the ACK message. It sends it off to the messenger
// for processing.
func (n *OpenBazaarNode) handleAckMessage(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_ACK {
		return errors.New("message is not type ACK")
	}
	ack := new(pb.AckMessage)
	if err := ptypes.UnmarshalAny(message.Payload, ack); err != nil {
		return err
	}
	err := n.repo.DB().Update(func(tx *gorm.DB) error {
		return n.messenger.ProcessACK(tx, ack)
	})
	if err != nil {
		return err
	}
	n.eventBus.Emit(&events.MessageACK{MessageID: ack.AckedMessageID})
	return nil
}

func (n *OpenBazaarNode) handleStoreMessage(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_STORE {
		return errors.New("message is not type STORE")
	}
	following, err := n.repo.PublicData().GetFollowing()
	if err != nil {
		return err
	}
	if !following.IsFollowing(from) {
		return errors.New("STORE message from peer that is not followed")
	}

	store := new(pb.StoreMessage)
	if err := ptypes.UnmarshalAny(message.Payload, store); err != nil {
		return err
	}

	var cids []cid.Cid
	for _, b := range store.Cids {
		cid, err := cid.Cast(b)
		if err != nil {
			return fmt.Errorf("store handler cid cast error: %s", err)
		}
		cids = append(cids, cid)
		if err := n.pin(path.Join(path.New("/ipfs"), cid.String())); err != nil {
			return fmt.Errorf("store handler error pinning file: %s", err)
		}
	}
	n.eventBus.Emit(&events.MessageStore{
		Peer: from,
		Cids: cids,
	})
	log.Infof("Received STORE message from %s", from)
	return nil
}

// isDuplicate checks if the message ID exists in the incoming messages database.
func (n *OpenBazaarNode) isDuplicate(message *pb.Message) bool {
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("id = ?", message.MessageID).First(&models.IncomingMessage{}).Error
	})
	return err == nil
}

// syncMessages listens for new connections to peers and checks to see if we have
// any outgoing messages for them. If so we send the messages over the direct
// connection.
func (n *OpenBazaarNode) syncMessages() {
	connectedSub, err := n.eventBus.Subscribe(&events.PeerConnected{})
	if err != nil {
		log.Error("Error subscribing to PeerConnected event: %s", err)
	}
	for {
		select {
		case event := <-connectedSub.Out():
			notif, ok := event.(*events.PeerConnected)
			if !ok {
				log.Error("syncMessages type assertion failed on PeerConnected")
				continue
			}
			var messages []models.OutgoingMessage
			err = n.repo.DB().View(func(tx *gorm.DB) error {
				return tx.Where("recipient = ?", notif.Peer.Pretty()).Find(&messages).Error
			})
			if err != nil && !gorm.IsRecordNotFoundError(err) {
				log.Error("syncMessages outgoing messages lookup error: %s", err)
				continue
			}
			for _, om := range messages {
				var message pb.Message
				if err := proto.Unmarshal(om.SerializedMessage, &message); err != nil {
					log.Error("syncMessages unmarshal error: %s", err)
					continue
				}
				recipient, err := peer.IDB58Decode(om.Recipient)
				if err != nil {
					log.Error("syncMessages peer decode error: %s", err)
					continue
				}
				go n.networkService.SendMessage(context.Background(), recipient, &message)
			}
		case <-n.shutdown:
			return
		}
	}
}

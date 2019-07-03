package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
)

// newMessageWithID returns a new *pb.Message with a random
// message ID and the correct sequence.
func prepMessage(tx *gorm.DB, peer peer.ID, messageType pb.Message_MessageType) (*pb.Message, error) {
	messageID := make([]byte, 20)
	rand.Read(messageID)

	parkedType, parkable := parkableMessages[messageType]

	var nseq uint32
	if parkable {
		seq := models.Sequence{
			PeerID: peer.Pretty(),
			MessageType: parkedType,
			Outgoing: true,
		}
		err := tx.Select("max(num)").Where("peer_id = ? AND message_type = ? AND outgoing = ?", peer.Pretty(), parkedType, true).First(&seq).Error
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			return nil, err
		}

		
		seq.Num++

		if err := tx.Save(seq).Error; err != nil {
			return nil, err
		}

		nseq = uint32(seq.Num)
	}

	return &pb.Message{
		MessageID:   hex.EncodeToString(messageID),
		MessageType: messageType,
		Sequence:    nseq,
	}, nil
}

// handleMessages is the main message handler for the node. It farms the
// processing out to the individual handlers.
func (n *OpenBazaarNode) handleMessages(from peer.ID, message *pb.Message) error {
	// Send the ACK message after we process it.
	defer n.sendAckMessage(from, message)

	// Duplicates we can just log. We still send the ACK here.
	if n.isDuplicate(message) {
		log.Debugf("Received duplicate message %s", message.MessageID)
		return nil
	}

	// Check if this message arrived out of order. If so we park it for
	// later processing.
	parked, err := n.maybeParkMessage(from, message)
	if err != nil {
		return err
	}

	if parked {
		log.Debugf("Parking message %s", message.MessageID)
		return nil
	}

	// Pass off to handlers.
	switch message.MessageType {
	case pb.Message_ACK:
		return n.handleAckMessage(from, message)
	case pb.Message_CHAT_MESSAGE:
		return n.handleChatMessage(from, message)
	case pb.Message_CHAT_READ:
		return n.handleReadMessage(from, message)
	case pb.Message_CHAT_TYPING:
		return n.handleTypingMessage(from, message)
	case pb.Message_FOLLOW:
		return n.handleFollowMessage(from, message)
	case pb.Message_UNFOLLOW:
		return n.handleUnfollowMessage(from, message)
	}
	return errors.New("unknown message type")
}

// sendAckMessage saves the incoming message ID in the database so we can
// check for duplicate messages later. Then it sends the ACK message to
// the remote peer.
func (n *OpenBazaarNode) sendAckMessage(to peer.ID, message *pb.Message) {
	if message.MessageType == pb.Message_ACK {
		return
	}
	err := n.repo.DB().Update(func(tx *gorm.DB) error {
		return tx.Save(&models.IncomingMessage{ID: message.MessageID}).Error
	})
	if err != nil {
		log.Errorf("Error saving incoming message ID to database: %s", err)
	}
	n.messenger.SendACK(message.MessageID, to)
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

// isDuplicate checks if the message ID exists in the incoming messages database.
func (n *OpenBazaarNode) isDuplicate(message *pb.Message) bool {
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("id = ?", message.MessageID).First(&models.IncomingMessage{}).Error
	})
	return err == nil
}

// maybeParkMessage checks if the message sequence number is ahead of
// last known number by more than one. If so it saves the message in
// the database for future processing.
func (n *OpenBazaarNode) maybeParkMessage(from peer.ID, message *pb.Message) (bool, error) {
	pmt, parkable := parkableMessages[message.MessageType]
	if !parkable { // Not parkable. Just return.
		return false, nil
	}

	parked := false
	err := n.repo.DB().Update(func(tx *gorm.DB) error {
		seq := models.Sequence{
			PeerID: from.Pretty(),
			MessageType: pmt,
			Outgoing: true,
		}
		err := tx.Select("max(num)").Where("peer_id = ? AND message_type = ? AND outgoing = ?", from.Pretty(), pmt, false).First(&seq).Error
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}

		// If the new sequence is more than one ahead of the last sequence
		// we know about, then we will park this message and wait for the
		// previous one to be discovered.
		if message.Sequence-uint32(seq.Num) > 1 {
			ser, err := proto.Marshal(message)
			if err != nil {
				return err
			}
			pm := &models.ParkedMessage{
				MessageType: pmt,
				Serialized:  ser,
			}
			if err := tx.Save(pm).Error; err != nil {
				return err
			}
			parked = true
			return nil
		}

		// Save with the new sequence.
		seq.Num = int(message.Sequence)
		return tx.Save(seq).Error
	})
	return parked, err
}

var parkableMessages = map[pb.Message_MessageType]models.ParkedMessageType {
	pb.Message_CHAT_MESSAGE: models.PmtChat,
	pb.Message_FOLLOW: models.PmtFollow,
	pb.Message_UNFOLLOW: models.PmtFollow,
}
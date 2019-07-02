package core

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
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

// isDuplicate checks if the message ID exists in the incoming messages database.
func (n *OpenBazaarNode) isDuplicate(message *pb.Message) bool {
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("id = ?", message.MessageID).First(&models.IncomingMessage{}).Error
	})
	return err == nil
}

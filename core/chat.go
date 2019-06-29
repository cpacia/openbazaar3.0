package core

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
)

func (n *OpenBazaarNode) SendChatMessage(to peer.ID, message, subject string, done chan<- struct{}) error {
	chatMsg := pb.ChatMessage{
		Message:   message,
		Subject:   subject,
		Timestamp: ptypes.TimestampNow(),
		Flag:      pb.ChatMessage_MESSAGE,
	}

	payload, err := ptypes.MarshalAny(&chatMsg)
	if err != nil {
		return err
	}

	msg := newMessageWithID()
	msg.MessageType = pb.Message_CHAT
	msg.Payload = payload

	chatModel, err := models.NewChatMessageFromProto(to, msg)
	if err != nil {
		return err
	}
	chatModel.Outgoing = true

	log.Debugf("Sending CHAT message to %s. MessageID: %s", to, msg.MessageID)
	return n.repo.DB().Update(func(tx *gorm.DB) error {
		if err := tx.Save(chatModel).Error; err != nil {
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, to, msg, done); err != nil {
			return err
		}

		return nil
	})
}

func (n *OpenBazaarNode) SendTypingMessage(ctx context.Context, to peer.ID) error {
	chatMsg := pb.ChatMessage{
		Flag: pb.ChatMessage_TYPING,
	}

	payload, err := ptypes.MarshalAny(&chatMsg)
	if err != nil {
		return err
	}

	msg := newMessageWithID()
	msg.MessageType = pb.Message_CHAT
	msg.Payload = payload

	// A Typing message is one of the rare messages that we don't care if it's
	// sent reliably so we don't need to send it with the Messenger. A simple
	// best effort direct message suffices.
	return n.networkService.SendMessage(ctx, to, msg)
}

// MarkChatMessagesAsRead will mark chat messages for the given subject as read locally
// and send the READ message to the remote peer. The READ message contains the last ID
// that was marked as read the remote peer will set that message and everything before
// it as read.
func (n *OpenBazaarNode) MarkChatMessagesAsRead(peer peer.ID, subject string) error {
	return n.repo.DB().Update(func(tx *gorm.DB)error {
		// Check unread count. If zero we can just exit.
		var unreadCount int
		if err := tx.Where("peer_id = ? AND read = ? AND subject = ?", peer.Pretty(), false, subject).Find(&models.ChatMessage{}).Count(&unreadCount).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}

		if unreadCount == 0 {
			return nil
		}

		// Load the last message
		lastMessage := models.ChatMessage{}
		if err := tx.Order("timestamp desc").Where("peer_id = ? AND read = ? AND subject = ?", peer.Pretty(), false, subject).First(&lastMessage).Error; err != nil {
			return err
		}

		// Update the local DB
		if err := tx.Model(&models.ChatMessage{}).Where("peer_id = ? AND subject = ?", peer.Pretty(), subject).UpdateColumn("read", true).Error; err != nil {
			return err
		}

		// Build and send the READ message back to the remote peer.
		// This is different than the ACK as the ACK is used for reliable
		// transport. The READ message is just to show the message was
		// read in the UI.
		readMsg := &pb.ChatMessage{
			Flag:    pb.ChatMessage_READ,
			Subject: subject,
			ReadID:  lastMessage.MessageID,
		}

		payload, err := ptypes.MarshalAny(readMsg)
		if err != nil {
			return err
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_CHAT
		msg.Payload = payload

		log.Debugf("Sending READ message to %s. MessageID: %s", peer, msg.MessageID)
		if err := n.messenger.ReliablySendMessage(tx, peer, msg, nil); err != nil {
			return err
		}

		return nil
	})
}

// GetChatConversations returns a list of conversations for the default subject ("")
// with some metadata included.
func (n *OpenBazaarNode) GetChatConversations() ([]models.ChatConversation, error) {
	var convos []models.ChatConversation
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		rows, err := tx.Raw("select distinct peer_id from chat_messages where subject='' order by timestamp desc;").Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		var ids []string
		for rows.Next() {
			var peerID string
			if err := rows.Scan(&peerID); err != nil {
				return err
			}
			ids = append(ids, peerID)
		}

		for _, peer := range ids {
			var message models.ChatMessage
			if err := tx.Order("timestamp desc").Where("peer_id", peer).Last(&message).Error; err != nil {
				return err
			}
			var count int
			if err := tx.Where("peer_id = ? AND read = ?", peer, false).Find(&models.ChatMessage{}).Count(&count).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
				return err
			}

			convo := models.ChatConversation{
				Last:      message.Message,
				PeerID:    peer,
				Outgoing:  message.Outgoing,
				Timestamp: message.Timestamp,
				Unread:    count,
			}
			convos = append(convos, convo)
		}
		return nil

	})
	if err != nil {
		return nil, err
	}
	return convos, nil
}

// GetChatMessagesByPeer returns a list of chat messages for a given peer ID.
func (n *OpenBazaarNode) GetChatMessagesByPeer(peer peer.ID) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("peer_id = ?", peer.Pretty()).Find(&messages).Error
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}
	return messages, nil
}

// GetChatMessagesBySubject returns a list of chat messages for a given subject.
func (n *OpenBazaarNode) GetChatMessagesBySubject(subject string) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := n.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("subject = ?", subject).Find(&messages).Error
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}
	return messages, nil
}

// handleChatMessage handles incoming chat messages from the network.
func (n *OpenBazaarNode) handleChatMessage(from peer.ID, message *pb.Message) error {
	defer n.messenger.SendACK(message.MessageID, from)

	chatMsg := new(pb.ChatMessage)
	if err := ptypes.UnmarshalAny(message.Payload, chatMsg); err != nil {
		return err
	}

	switch chatMsg.Flag {
	case pb.ChatMessage_MESSAGE:
		log.Infof("Received CHAT message from %s. MessageID: %s", from, message.MessageID)
		return n.repo.DB().Update(func(tx *gorm.DB) error {
			incomingMsg, err := models.NewChatMessageFromProto(from, message)
			if err != nil {
				return err
			}

			// Save the incoming message to the DB
			if err := tx.Save(incomingMsg).Error; err != nil {
				return err
			}

			// TODO: send chat message notification to the UI

			return nil
		})
	case pb.ChatMessage_READ:
		log.Infof("Received READ message from %s. MessageID: %s", from, message.MessageID)
		return n.repo.DB().Update(func(tx *gorm.DB) error {
			// Load the message with the provided ID
			var message models.ChatMessage
			if err := tx.Where("message_id", chatMsg.ReadID).First(&message).Error; err != nil {
				return err
			}

			// Update all unread messages before the given message ID.
			if err := tx.Model(&models.ChatMessage{}).Where("peer_id = ? AND read = ? AND subject = ? AND timestamp <= ?", from.Pretty(), false, chatMsg.Subject, message.Timestamp).UpdateColumn("read", true).Error; err != nil {
				return err
			}

			// TODO: send read notification to UI
			return nil
		})
	case pb.ChatMessage_TYPING:
		// TODO: send typing notification to API
		return nil

	default:
		return errors.New("unknown chat message flag")

	}
}

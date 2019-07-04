package core

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

// SendChatMessage sends a chat message to the given peer. The message is sent
// reliably using the messenger in a separate goroutine so this method will
// not block during the send. The done chan will be closed when the sending is
// complete if you need this information.
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

	return n.repo.DB().Update(func(tx *gorm.DB) error {
		var prev models.ChatMessage
		if err := tx.Order("timestamp desc").Where("peer_id = ? AND outgoing = ?", to.Pretty(), true).Last(&prev).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_CHAT
		msg.Payload = payload
		msg.Sequence = uint32(prev.Sequence + 1)

		chatModel, err := models.NewChatMessageFromProto(to, msg)
		if err != nil {
			return err
		}
		chatModel.Outgoing = true
		chatModel.Sequence = prev.Sequence + 1

		if err := tx.Save(chatModel).Error; err != nil {
			return err
		}

		log.Debugf("Sending CHAT message to %s. MessageID: %s", to, msg.MessageID)
		return n.messenger.ReliablySendMessage(tx, to, msg, done)
	})
}

// SendTypingMessage sends the typing message to the remote peer which
// the UI can then using to display that the peer is typing. This message
// is only sent using direct messaging and on a best effort basis. This
// means it's not guaranteed to make it to the remote peer.
func (n *OpenBazaarNode) SendTypingMessage(to peer.ID, subject string) error {
	chatMsg := pb.ChatMessage{
		Flag:    pb.ChatMessage_TYPING,
		Subject: subject,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	return n.networkService.SendMessage(ctx, to, msg)
}

// MarkChatMessagesAsRead will mark chat messages for the given subject as read locally
// and send the READ message to the remote peer. The READ message contains the last ID
// that was marked as read the remote peer will set that message and everything before
// it as read.
func (n *OpenBazaarNode) MarkChatMessagesAsRead(peer peer.ID, subject string) error {
	return n.repo.DB().Update(func(tx *gorm.DB) error {
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
		return n.messenger.ReliablySendMessage(tx, peer, msg, nil)
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
			if err := tx.Order("timestamp desc").Where("peer_id = ?", peer).Last(&message).Error; err != nil {
				return err
			}
			var unreadCount int
			if err := tx.Where("peer_id = ? AND read = ? AND subject = ? AND outgoing = ?", peer, false, "", false).Find(&models.ChatMessage{}).Count(&unreadCount).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
				return err
			}

			convo := models.ChatConversation{
				Last:      message.Message,
				PeerID:    peer,
				Outgoing:  message.Outgoing,
				Timestamp: message.Timestamp,
				Unread:    unreadCount,
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
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	chatMsg := new(pb.ChatMessage)
	if err := ptypes.UnmarshalAny(message.Payload, chatMsg); err != nil {
		return err
	}

	switch chatMsg.Flag {
	case pb.ChatMessage_MESSAGE:
		log.Infof("Received CHAT message from %s. MessageID: %s", from, message.MessageID)
		incomingMsg, err := models.NewChatMessageFromProto(from, message)
		if err != nil {
			return err
		}
		err = n.repo.DB().Update(func(tx *gorm.DB) error {
			// Save the incoming message to the DB
			return tx.Save(incomingMsg).Error
		})
		if err != nil {
			return err
		}
		n.eventBus.Emit(incomingMsg.ToChatNotification())
		return nil
	case pb.ChatMessage_READ:
		log.Infof("Received READ message from %s. MessageID: %s. ReadID: %s", from, message.MessageID, chatMsg.ReadID)
		err := n.repo.DB().Update(func(tx *gorm.DB) error {
			// Load the message with the provided ID
			var chmsg models.ChatMessage
			if err := tx.Where("message_id = ?", chatMsg.ReadID).Find(&chmsg).Error; err != nil {
				return err
			}

			// Update all unread messages before the given message ID.
			if err := tx.Model(&models.ChatMessage{}).Where("peer_id = ? AND read = ? AND subject = ? AND timestamp <= ?", from.Pretty(), false, chatMsg.Subject, chmsg.Timestamp).UpdateColumn("read", true).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		n.eventBus.Emit(&events.ChatReadNotification{
			Subject:   chatMsg.Subject,
			PeerID:    from.String(),
			MessageID: chatMsg.ReadID,
		})
		return nil
	case pb.ChatMessage_TYPING:
		n.eventBus.Emit(&events.ChatTypingNotification{
			MessageID: message.MessageID,
			PeerID:    from.Pretty(),
			Subject:   chatMsg.Subject,
		})
		return nil

	default:
		return errors.New("unknown chat message flag")

	}
}

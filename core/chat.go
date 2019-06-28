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

	log.Debugf("Sending CHAT message to %s. MessageID: %s", to, msg.MessageID)
	return n.repo.DBUpdate(func(tx *gorm.DB) error {
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

func (n *OpenBazaarNode) handleChatMessage(from peer.ID, message *pb.Message) error {
	defer n.messenger.SendACK(message.MessageID, from)

	chatMsg := new(pb.ChatMessage)
	if err := ptypes.UnmarshalAny(message.Payload, chatMsg); err != nil {
		return err
	}

	switch chatMsg.Flag {
	case pb.ChatMessage_MESSAGE:
		log.Debugf("Received CHAT message from %s", from)
		return n.repo.DBUpdate(func(tx *gorm.DB) error {
			incomingMsg, err := models.NewChatMessageFromProto(from, message)
			if err != nil {
				return err
			}
			// Save the incoming message to the DB
			if err := tx.Save(incomingMsg).Error; err != nil {
				return err
			}

			// Build and send the READ message back to the remote peer.
			// This is different than the ACK as the ACK is used for reliable
			// transport. The READ message is just to show the message was
			// read in the UI.
			readMsg := &pb.ChatMessage{
				Flag: pb.ChatMessage_READ,
			}

			payload, err := ptypes.MarshalAny(readMsg)
			if err != nil {
				return err
			}

			msg := newMessageWithID()
			msg.MessageType = pb.Message_CHAT
			msg.Payload = payload

			log.Debugf("Sending READ message to %s. MessageID: %s", from, msg.MessageID)
			if err := n.messenger.ReliablySendMessage(tx, from, msg, nil); err != nil {
				return err
			}

			return nil
		})
	case pb.ChatMessage_READ:
		log.Debugf("Received READ message from %s", from)
		return n.repo.DBUpdate(func(tx *gorm.DB) error {
			msg := models.ChatMessage{
				MessageID: message.MessageID,
			}
			if err := tx.Model(&msg).Update("read", true).Error; err != nil {
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

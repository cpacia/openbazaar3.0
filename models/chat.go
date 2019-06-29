package models

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

type ChatMessage struct {
	MessageID string    `gorm:"primary_key"`
	PeerID    string    `gorm:"index"`
	Subject   string    `gorm:"index"`
	Timestamp time.Time `gorm:"index"`
	Read      bool      `gorm:"index"`
	Outgoing  bool
	Message   string
}

func NewChatMessageFromProto(peerID peer.ID, msg *pb.Message) (*ChatMessage, error) {
	if msg.MessageType != pb.Message_CHAT {
		return nil, errors.New("cannot convert non-CHAT message type")
	}

	chtMsg := new(pb.ChatMessage)
	if err := ptypes.UnmarshalAny(msg.Payload, chtMsg); err != nil {
		return nil, err
	}

	return &ChatMessage{
		MessageID: msg.MessageID,
		PeerID:    peerID.Pretty(),
		Message:   chtMsg.Message,
		Subject:   chtMsg.Subject,
		Timestamp: time.Unix(chtMsg.Timestamp.Seconds, int64(chtMsg.Timestamp.Nanos)),
	}, nil
}

func (cm *ChatMessage) GetPeerID() (peer.ID, error) {
	return peer.IDFromString(cm.PeerID)
}

type ChatConversation struct {
	PeerID    string    `json:"peerID"`
	Unread    int       `json:"unread"`
	Last      string    `json:"lastMessage"`
	Timestamp time.Time `json:"timestamp"`
	Outgoing  bool      `json:"outgoing"`
}

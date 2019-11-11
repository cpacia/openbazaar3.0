package models

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

type ChatMessage struct {
	MessageID string    `gorm:"primary_key" json:"messageID"`
	PeerID    string    `gorm:"index" json:"peerID"`
	OrderID   string    `gorm:"index" json:"subject"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
	Read      bool      `gorm:"index" json:"read"`
	Outgoing  bool      `json:"outgoing"`
	Message   string    `json:"message"`
	Sequence  int
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
		OrderID:   chtMsg.OrderID,
		Timestamp: time.Unix(chtMsg.Timestamp.Seconds, int64(chtMsg.Timestamp.Nanos)),
		Sequence:  int(msg.Sequence),
	}, nil
}

func (cm *ChatMessage) GetPeerID() (peer.ID, error) {
	return peer.IDB58Decode(cm.PeerID)
}

func (cm *ChatMessage) ToChatNotification() *events.ChatMessageNotification {
	return &events.ChatMessageNotification{
		MessageID: cm.MessageID,
		Timestamp: cm.Timestamp,
		PeerID:    cm.PeerID,
		OrderID:   cm.OrderID,
		Outgoing:  cm.Outgoing,
		Read:      cm.Read,
		Message:   cm.Message,
	}
}

type ChatConversation struct {
	PeerID    string    `json:"peerID"`
	Unread    int       `json:"unread"`
	Last      string    `json:"lastMessage"`
	Timestamp time.Time `json:"timestamp"`
	Outgoing  bool      `json:"outgoing"`
}

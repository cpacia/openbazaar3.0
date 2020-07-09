package models

import (
	"encoding/json"
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p-core/peer"
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
	return peer.Decode(cm.PeerID)
}

func (cm *ChatMessage) ToChatEvent() *events.ChatMessage {
	return &events.ChatMessage{
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

type ChannelMessage struct {
	PeerID    string    `json:"peerID"`
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Cid       string    `json:"cid"`
}

type Channel struct {
	Topic       string `gorm:"primary_key"`
	LastMessage time.Time
	Head        []byte
}

func (c *Channel) GetHead() ([]cid.Cid, error) {
	var ret []cid.Cid
	if len(c.Head) == 0 {
		return ret, nil
	}
	if err := json.Unmarshal(c.Head, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Channel) SetHead(ids []cid.Cid) error {
	out, err := json.MarshalIndent(ids, "", "    ")
	if err != nil {
		return err
	}
	c.Head = out
	return nil
}

func (c *Channel) UpdateHead(nd ipld.Node) error {
	ids, err := c.GetHead()
	if err != nil {
		return err
	}
	idMap := make(map[cid.Cid]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	for _, link := range nd.Links() {
		if idMap[link.Cid] {
			delete(idMap, link.Cid)
		}
	}

	newState := make([]cid.Cid, 0, len(idMap)+1)
	if !idMap[nd.Cid()] {
		newState = append(newState, nd.Cid())
	}
	for id := range idMap {
		newState = append(newState, id)
		if len(idMap) >= 49 {
			break
		}
	}

	out, err := json.MarshalIndent(newState, "", "    ")
	if err != nil {
		return err
	}
	c.Head = out
	return nil
}

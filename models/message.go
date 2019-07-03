package models

import (
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"time"
)

// OutgoingMessage represents a message that we've sent to another
// peer. It will remain in the database until the remote peer ACKs
// the message.
type OutgoingMessage struct {
	ID                string `gorm:"primary_key"`
	Recipient         string `gorm:"index"`
	SerializedMessage []byte
	Timestamp         time.Time
	LastAttempt       time.Time
}

func (m *OutgoingMessage) Message() (*pb.Message, error) {
	msg := new(pb.Message)
	if err := proto.Unmarshal(m.SerializedMessage, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// IncomingMessage represents a message that we've received. We store
// all received message IDs in the database so we can tell when we've
// received a duplicate.
type IncomingMessage struct {
	ID string `gorm:"primary_key"`
}

// ParkedMessageType denote the type of message that was parked.
// Some message types, such as follow/unfollow we want to track
// the sequence across both types. Hence we use a separate type
// here rather than a pb.Message_Type.
type ParkedMessageType string

const (
	// PmtChat denotes a parked chat message.
	PmtChat ParkedMessageType = "CHAT"
	// PmtFollow denotes a parked follow message.
	PmtFollow ParkedMessageType = "FOLLOW"
	// PmtOrder denates a parked order message.
	PmtOrder ParkedMessageType = "ORDER"
)

// ParkedMessage represents a message that has been stored for later
// processing due to it containing a later sequence than the last
// processed message.
type ParkedMessage struct {
	MessageType ParkedMessageType `gorm:"index"`
	Serialized  []byte
}

// Sequence represents a sequence number used for constructing
// message. Some messages, like follow/unfollow or order messages
// need to be processed in a certain order. By tracking sequence
// numbers we can communicate the order in which message should
// be processed.
type Sequence struct {
	PeerID      string            `gorm:"index"`
	MessageType ParkedMessageType `gorm:"index"`
	Outgoing    bool
	Num         int
}

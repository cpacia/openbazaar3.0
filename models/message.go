package models

import (
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"time"
)

type OutgoingMessage struct {
	ID                string `gorm:"primary_key"`
	Recipient         string `gorm:"index"`
	SerializedMessage []byte
	LastAttempt       time.Time
}

func (m *OutgoingMessage) Message() (*pb.Message, error) {
	msg := new(pb.Message)
	if err := proto.Unmarshal(m.SerializedMessage, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

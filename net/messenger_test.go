package net

import (
	"context"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"sync"
	"testing"
)

func TestMessenger(t *testing.T) {
	mocknet, err := mocknet.FullMeshLinked(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	service1 := NewNetworkService(mocknet.Hosts()[0], NewBanManager(nil), true)
	service2 := NewNetworkService(mocknet.Hosts()[1], NewBanManager(nil), true)

	db1, err := repo.MockDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()

	db2, err := repo.MockDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	ctx, cancel := context.WithCancel(context.Background())
	messenger1 := &Messenger{service1, db1, ctx, cancel, sync.RWMutex{}}
	messenger2 := &Messenger{service2, db2, ctx, cancel, sync.RWMutex{}}

	ch := make(chan struct{})
	service2.RegisterHandler(pb.Message_PING, func(p peer.ID, msg *pb.Message) error {
		ch <- struct{}{}
		return nil
	})

	ch2 := make(chan struct{})
	service1.RegisterHandler(pb.Message_ACK, func(p peer.ID, msg *pb.Message) error {
		err := db1.Update(func(tx *gorm.DB) error {
			return messenger1.ProcessACK(tx, msg)
		})
		if err != nil {
			t.Error(err)
		}

		ch2 <- struct{}{}
		return nil
	})

	done := make(chan struct{})
	db1.Update(func(tx *gorm.DB) error {
		messenger1.ReliablySendMessage(tx, service2.host.ID(), &pb.Message{MessageID: "abc", MessageType: pb.Message_PING}, done)
		return nil
	})

	var messages []models.OutgoingMessage
	err = messenger1.db.View(func(tx *gorm.DB) error {
		return tx.Find(&messages).Error
	})
	if err != nil {
		t.Error(err)
	}

	if len(messages) != 1 {
		t.Error("Failed to delete ACKed message from the db")
	}

	<-done
	<-ch

	messenger2.SendACK("abc", service1.host.ID())
	<-ch2

	var messages2 []models.OutgoingMessage
	err = messenger1.db.View(func(tx *gorm.DB) error {
		return tx.Find(&messages2).Error
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		t.Error(err)
	}

	if len(messages2) > 0 {
		t.Error("Failed to delete ACKed message from the db")
	}
}

func TestMessenger_retryAllMessages(t *testing.T) {
	mocknet, err := mocknet.FullMeshLinked(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	service1 := NewNetworkService(mocknet.Hosts()[0], NewBanManager(nil), true)
	service2 := NewNetworkService(mocknet.Hosts()[1], NewBanManager(nil), true)

	db1, err := repo.MockDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()

	ctx, cancel := context.WithCancel(context.Background())
	messenger := &Messenger{service1, db1, ctx, cancel, sync.RWMutex{}}

	err = db1.Update(func(tx *gorm.DB) error {
		ping := &pb.Message{
			MessageType: pb.Message_PING,
			MessageID:   "abc",
		}
		ser, err := proto.Marshal(ping)
		if err != nil {
			return err
		}
		return tx.Save(&models.OutgoingMessage{
			ID:                "abc",
			Recipient:         service2.host.ID().Pretty(),
			SerializedMessage: ser,
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	messenger.retryAllMessages()
	ch := make(chan struct{})
	service2.RegisterHandler(pb.Message_PING, func(p peer.ID, msg *pb.Message) error {
		ch <- struct{}{}
		return nil
	})

	<-ch
}

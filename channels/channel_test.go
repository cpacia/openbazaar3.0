package channels_test

import (
	"context"
	"github.com/cpacia/openbazaar3.0/channels"
	"github.com/cpacia/openbazaar3.0/core"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"os"
	"path"
	"testing"
	"time"
)

func TestChannels(t *testing.T) {
	mn, err := core.NewMocknet(3)
	if err != nil {
		t.Fatal(err)
	}
	defer mn.TearDown()

	db0, err := ffsqlite.NewFFMemoryDB(path.Join(os.TempDir(), "channel_test", "0"))
	if err != nil {
		t.Fatal(err)
	}
	defer db0.Close()
	err = db0.Update(func(tx database.Tx) error {
		return tx.Migrate(&models.Channel{})
	})
	if err != nil {
		t.Fatal(err)
	}

	db1, err := ffsqlite.NewFFMemoryDB(path.Join(os.TempDir(), "channel_test", "1"))
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()
	err = db1.Update(func(tx database.Tx) error {
		return tx.Migrate(&models.Channel{})
	})
	if err != nil {
		t.Fatal(err)
	}

	bus0 := events.NewBus()
	channel0, err := channels.NewChannel("general", mn.Nodes()[1].IPFSNode(), nil, bus0, db0)
	if err != nil {
		t.Fatal(err)
	}
	defer channel0.Close()

	sub0, err := bus0.Subscribe(&events.ChannelBootstrapped{})
	if err != nil {
		t.Fatal(err)
	}

	bus1 := events.NewBus()
	channel1, err := channels.NewChannel("general", mn.Nodes()[2].IPFSNode(), nil, bus1, db0)
	if err != nil {
		t.Fatal(err)
	}
	defer channel1.Close()

	sub1, err := bus1.Subscribe(&events.ChannelBootstrapped{})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on bootstrap0")
	case <-sub0.Out():
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on bootstrap1")
	case <-sub1.Out():
	}

	sub2, err := bus1.Subscribe(&events.ChannelMessage{})
	if err != nil {
		t.Fatal(err)
	}

	sub3, err := bus0.Subscribe(&events.ChannelMessage{})
	if err != nil {
		t.Fatal(err)
	}

	if err := channel0.Publish(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on message")
	case event := <-sub2.Out():
		msg := event.(*events.ChannelMessage)
		if msg.Topic != channel0.Topic() {
			t.Errorf("Expected topic %s, got %s", channel0.Topic(), msg.Topic)
		}
		if msg.Message != "test" {
			t.Errorf("Expected message %s, got %s", "test", msg.Message)
		}
		if msg.PeerID != mn.Nodes()[1].Identity().Pretty() {
			t.Errorf("Expected peerID %s, got %s", mn.Nodes()[1].Identity().Pretty(), msg.PeerID)
		}
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on message")
	case event := <-sub3.Out():
		msg := event.(*events.ChannelMessage)
		if msg.Topic != channel0.Topic() {
			t.Errorf("Expected topic %s, got %s", channel0.Topic(), msg.Topic)
		}
		if msg.Message != "test" {
			t.Errorf("Expected message %s, got %s", "test", msg.Message)
		}
		if msg.PeerID != mn.Nodes()[1].Identity().Pretty() {
			t.Errorf("Expected peerID %s, got %s", mn.Nodes()[1].Identity().Pretty(), msg.PeerID)
		}
	}

	if err := channel1.Publish(context.Background(), "test2"); err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on message")
	case event := <-sub2.Out():
		msg := event.(*events.ChannelMessage)
		if msg.Topic != channel0.Topic() {
			t.Errorf("Expected topic %s, got %s", channel0.Topic(), msg.Topic)
		}
		if msg.Message != "test2" {
			t.Errorf("Expected message %s, got %s", "test2", msg.Message)
		}
		if msg.PeerID != mn.Nodes()[2].Identity().Pretty() {
			t.Errorf("Expected peerID %s, got %s", mn.Nodes()[2].Identity().Pretty(), msg.PeerID)
		}
	}

	select {
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on message")
	case event := <-sub3.Out():
		msg := event.(*events.ChannelMessage)
		if msg.Topic != channel0.Topic() {
			t.Errorf("Expected topic %s, got %s", channel0.Topic(), msg.Topic)
		}
		if msg.Message != "test2" {
			t.Errorf("Expected message %s, got %s", "test2", msg.Message)
		}
		if msg.PeerID != mn.Nodes()[2].Identity().Pretty() {
			t.Errorf("Expected peerID %s, got %s", mn.Nodes()[2].Identity().Pretty(), msg.PeerID)
		}
	}

	msgs, err := channel0.Messages(context.Background(), nil, -1)
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages got %d", len(msgs))
	}

	if msgs[0].Message != "test2" {
		t.Errorf("Expected message %s, got %s", "test2", msgs[0].Message)
	}

	if msgs[1].Message != "test" {
		t.Errorf("Expected message %s, got %s", "test", msgs[1].Message)
	}
}

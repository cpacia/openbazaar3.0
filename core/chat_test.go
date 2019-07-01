package core

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"testing"
)

func TestOpenBazaarNode_SendChatMessage(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}
	defer node.repo.DestroyRepo()

	p, err := peer.IDB58Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}

	var (
		message = "hola"
		subject = "newsubject"
	)

	done := make(chan struct{})
	if err := node.SendChatMessage(p, message, subject, done); err != nil {
		t.Fatal(err)
	}
	<- done

	var messages []models.ChatMessage
	err = node.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Find(&messages).Error
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 1 {
		t.Errorf("Returned incorrect number of messages from db. Expected %d, got %d", 1, len(messages))
	}

	if messages[0].Message != message {
		t.Errorf("Returned incorrect message from db. Expected %s, got %s", message, messages[0].Message)
	}

	if messages[0].Subject != subject {
		t.Errorf("Returned incorrect subject from db. Expected %s, got %s", subject, messages[0].Subject)
	}

	if messages[0].MessageID == "" {
		t.Error("Message ID is empty string")
	}

	if messages[0].Outgoing != true {
		t.Error("Message not marked as outgoing")
	}

	if messages[0].Read != false {
		t.Error("Message incorrectly marked as read")
	}

	if messages[0].PeerID != p.Pretty() {
		t.Errorf("Returned incorrect recipient ID. Expected %s, got %s", p.Pretty(), messages[0].PeerID)
	}
}

func TestOpenBazaarNode_SendTypingMessage(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	sub, err := network.Nodes()[1].eventBus.Subscribe(&events.ChatTypingNotification{})
	if err != nil {
		t.Fatal(err)
	}

	subject := "typing test"
	if err := network.Nodes()[0].SendTypingMessage(network.Nodes()[1].Identity(), subject); err != nil {
		t.Fatal(err)
	}

	event := <-sub.Out()

	notif, ok := event.(*events.ChatTypingNotification)
	if !ok {
		t.Fatal("Failed to type assert ChatTypingNotification")
	}

	if notif.Subject != subject {
		t.Errorf("Received incorrect subject. Expected %s, got %s", subject, notif.Subject)
	}

	if notif.PeerID != network.Nodes()[0].Identity().Pretty() {
		t.Errorf("Received incorrect peer ID. Expected %s, got %s", network.Nodes()[0].Identity().Pretty(), notif.PeerID)
	}
}

func TestOpenBazaarNode_MarkChatMessagesAsRead(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	sub, err := network.Nodes()[1].eventBus.Subscribe(&events.ChatMessageNotification{})
	if err != nil {
		t.Fatal(err)
	}

	var (
		subject = "advice"
		message = "abolish the state"
	)
	if err := network.Nodes()[0].SendChatMessage(network.Nodes()[1].Identity(), message, subject, nil); err != nil {
		t.Fatal(err)
	}

	event := <-sub.Out()
	notif, ok := event.(*events.ChatMessageNotification)
	if !ok {
		t.Fatal("Failed to type assert ChatMessageNotification")
	}

	if notif.Message != message {
		t.Errorf("Received incorrect message. Expected %s, got %s", message, notif.Message)
	}

	if notif.Subject != subject {
		t.Errorf("Received incorrect subject. Expected %s, got %s", subject, notif.Subject)
	}

	if notif.PeerID != network.Nodes()[0].Identity().Pretty() {
		t.Errorf("Received incorrect peer ID. Expected %s, got %s", network.Nodes()[0].Identity().Pretty(), notif.PeerID)
	}

	sub2, err := network.Nodes()[0].eventBus.Subscribe(&events.ChatReadNotification{})
	if err != nil {
		t.Fatal(err)
	}

	if err := network.Nodes()[1].MarkChatMessagesAsRead(network.Nodes()[0].Identity(), notif.Subject); err != nil {
		t.Fatal(err)
	}

	event2 := <-sub2.Out()
	notif2, ok := event2.(*events.ChatReadNotification)
	if !ok {
		t.Fatal("Failed to type assert ChatReadNotification")
	}

	if notif2.MessageID != notif.MessageID {
		t.Errorf("Read message ID doesn't match original message ID. Got %s, expected %s", notif2.MessageID, notif.MessageID)
	}

	if notif2.Subject != notif.Subject {
		t.Errorf("Received incorrect subject. Expected %s, got %s", subject, notif2.Subject)
	}

	if notif2.PeerID != network.Nodes()[1].Identity().Pretty() {
		t.Errorf("Received incorrect peer ID. Expected %s, got %s", network.Nodes()[1].Identity().Pretty(), notif2.PeerID)
	}

	var (
		chatMessage1 models.ChatMessage
		chatMessage2 models.ChatMessage
	)
	err = network.Nodes()[0].repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("message_id = ?", notif2.MessageID).First(&chatMessage1).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	err = network.Nodes()[1].repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("message_id = ?", notif.MessageID).First(&chatMessage2).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if !chatMessage1.Read {
		t.Error("Node 0 failed to mark chat message as read in database")
	}
	if !chatMessage2.Read {
		t.Error("Node 1 failed to mark chat message as read in database")
	}
}
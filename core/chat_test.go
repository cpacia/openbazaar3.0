package core

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	peer "github.com/libp2p/go-libp2p-peer"
	"testing"
	"time"
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
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	var messages []models.ChatMessage
	err = node.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Find(&messages).Error
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

	var event interface{}
	select {
	case event = <-sub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

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
	// Send message from 0 to 1
	if err := network.Nodes()[0].SendChatMessage(network.Nodes()[1].Identity(), message, subject, nil); err != nil {
		t.Fatal(err)
	}

	// Wait for 1 to receive the message.
	var event interface{}
	select {
	case event = <-sub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
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

	// Node 1 mark as read.
	if err := network.Nodes()[1].MarkChatMessagesAsRead(network.Nodes()[0].Identity(), notif.Subject); err != nil {
		t.Fatal(err)
	}

	// Wait for node 0 to receive the read notification.
	var event2 interface{}
	select {
	case event2 = <-sub2.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
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
	err = network.Nodes()[0].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("message_id = ?", notif2.MessageID).First(&chatMessage1).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("message_id = ?", notif.MessageID).First(&chatMessage2).Error
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

func TestOpenBazaarNode_GetChat(t *testing.T) {
	network, err := NewMocknet(4)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	var (
		firstMessage = "hola"
		lastMessage  = "hi again"
	)

	for _, node := range network.Nodes()[1:] {
		if err := network.Nodes()[0].SendChatMessage(node.Identity(), firstMessage, "", nil); err != nil {
			t.Fatal(err)
		}
		if err := network.Nodes()[0].SendChatMessage(node.Identity(), lastMessage, "", nil); err != nil {
			t.Fatal(err)
		}
	}

	convos, err := network.Nodes()[0].GetChatConversations()
	if err != nil {
		t.Fatal(err)
	}

	if len(convos) != 3 {
		t.Errorf("Expected 3 conversations got %d", len(convos))
	}

	for _, convo := range convos {
		if convo.Outgoing != true {
			t.Error("Outgoing bool is false")
		}
		if convo.Last != lastMessage {
			t.Errorf("Received incorrect last message. Expected %s, got %s", lastMessage, convo.Last)
		}
		if convo.Unread != 0 {
			t.Error("Non-zero unread incoming messages")
		}
	}

	for _, node := range network.Nodes()[1:] {
		messages, err := network.Nodes()[0].GetChatMessagesByPeer(node.Identity())
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 2 {
			t.Errorf("Expected 2 chat messages got %d", len(messages))
		}

		if messages[0].Read {
			t.Errorf("Message0 to peer %s is marked read when it should not be", node.Identity())
		}

		if messages[1].Read {
			t.Errorf("Message1 is to peer %s marked read when it should not be", node.Identity())
		}

		if messages[0].Message != firstMessage {
			t.Errorf("Incorrect first message. Expected %s, got %s", firstMessage, messages[0].Message)
		}
		if messages[1].Message != lastMessage {
			t.Errorf("Incorrect first message. Expected %s, got %s", firstMessage, messages[0].Message)
		}
		if messages[0].Outgoing != true {
			t.Error("Message0 is not set to outgoing when it should be")
		}
		if messages[1].Outgoing != true {
			t.Error("Message1 is not set to outgoing when it should be")
		}
		if messages[0].PeerID != node.Identity().Pretty() {
			t.Errorf("Message0 peer ID does not match peer. Expected %s, got %s", node.Identity().Pretty(), messages[0].PeerID)
		}
		if messages[1].PeerID != node.Identity().Pretty() {
			t.Errorf("Message1 peer ID does not match peer. Expected %s, got %s", node.Identity().Pretty(), messages[1].PeerID)
		}
	}

	messages, err := network.Nodes()[0].GetChatMessagesBySubject("")
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 6 {
		t.Errorf("Expected 6 messages, got %d", len(messages))
	}

	for _, message := range messages {
		if message.Read {
			t.Error("Message is marked as read when it should not be")
		}
		if !message.Outgoing {
			t.Error("Message is not set to outgoing when it should be")
		}
	}
}

func TestOpenBazaarNode_ChatSequence(t *testing.T) {
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
	if err := node.SendChatMessage(p, message, subject, nil); err != nil {
		t.Fatal(err)
	}
	if err := node.SendChatMessage(p, message, subject, nil); err != nil {
		t.Fatal(err)
	}
	if err := node.SendChatMessage(p, message, subject, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	var messages []models.ChatMessage
	err = node.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Find(&messages).Error
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 3 {
		t.Fatalf("Incorrect number of chat messages. Expected 3 got %d", len(messages))
	}

	for i, c := range messages {
		if c.Sequence != i+1 {
			t.Errorf("Incorrect sequence number. Expected %d, got %d", i+1, c.Sequence)
		}
	}
}

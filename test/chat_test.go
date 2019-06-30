package test

import (
	"fmt"
	"github.com/cpacia/openbazaar3.0/events"
	"testing"
)

func TestChat(t *testing.T) {
	t.Parallel()

	network, err := NewTestNetwork(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	sub, err := network.Nodes[0].SubscribeEvent(&events.ChatReadNotification{})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	messageText := "Hello there!"
	done := make(chan struct{})
	if err := network.Nodes[0].SendChatMessage(network.Nodes[1].Identity(), messageText, "", done); err != nil {
		t.Error(err)
	}
	<-done

	if err := network.Nodes[1].MarkChatMessagesAsRead(network.Nodes[0].Identity(), ""); err != nil {
		t.Error(err)
	}

	<-sub.Out()

	messages0, err := network.Nodes[0].GetChatMessagesBySubject("")
	if err != nil {
		t.Error(err)
	}
	messages1, err := network.Nodes[1].GetChatMessagesBySubject("")
	if err != nil {
		t.Error(err)
	}

	if len(messages0) != 1 {
		t.Errorf("Incorrect number of messages for peer 0. Expected 1 got %d", len(messages0))
	}

	if len(messages1) != 1 {
		t.Errorf("Incorrect number of messages for peer 1. Expected 1 got %d", len(messages1))
	}

	if !messages0[0].Read {
		t.Errorf("Node 0 failed to mark message as read")
	}

	if !messages1[0].Read {
		t.Errorf("Node 1 failed to mark message as read")
	}

	if messages0[0].Message != messageText {
		t.Errorf("Node 0 failed to save correct message. Expected %s, got %s", messageText, messages0[0].Message)
	}

	if messages1[0].Message != messageText {
		t.Errorf("Node 1 failed to save correct message. Expected %s, got %s", messageText, messages0[0].Message)
	}

	if messages0[0].Subject != "" {
		t.Errorf("Node 0 failed to save correct subject. Expected %s, got %s", "", messages0[0].Message)
	}

	if messages1[0].Subject != "" {
		t.Errorf("Node 1 failed to save correct subject. Expected %s, got %s", "", messages0[0].Message)
	}

	convos, err := network.Nodes[0].GetChatConversations()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(convos)
}

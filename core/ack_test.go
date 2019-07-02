package core

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/jinzhu/gorm"
	"testing"
)

func Test_SendAndReceiveAcks(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	sub, err := network.Nodes()[1].SubscribeEvent(&events.ChatMessageNotification{})
	if err != nil {
		t.Fatal(err)
	}

	if err := network.Nodes()[0].SendChatMessage(network.nodes[1].Identity(), "hola", "", nil); err != nil {
		t.Fatal(err)
	}

	<-sub.Out()

	var chatMessages []models.ChatMessage
	err = network.Nodes()[1].repo.DB().View(func(tx *gorm.DB) error {
		return tx.Model(&models.ChatMessage{}).Find(&chatMessages).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(chatMessages) != 1 {
		t.Fatalf("Incorrect number of messages. Expected %d, got %d", 1, len(chatMessages))
	}

	sub2, err := network.Nodes()[0].SubscribeEvent(&events.MessageACK{})
	if err != nil {
		t.Fatal(err)
	}

	network.Nodes()[1].sendAckMessage(chatMessages[0].MessageID, network.Nodes()[0].Identity())

	var incomingMessages []models.IncomingMessage
	err = network.Nodes()[1].repo.DB().View(func(tx *gorm.DB) error {
		return tx.Model(&models.IncomingMessage{}).Find(&incomingMessages).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(incomingMessages) != 1 {
		t.Fatalf("Incorrect number of incoming messages. Expected %d, got %d", 1, len(incomingMessages))
	}

	if incomingMessages[0].ID != chatMessages[0].MessageID {
		t.Errorf("Saved incorrect incoming message ID. Expected %s, got %s", chatMessages[0].MessageID, incomingMessages[0].ID)
	}

	event := <-sub2.Out()

	notif, ok := event.(*events.MessageACK)
	if !ok {
		t.Fatalf("Event type conversion failed")
	}

	if notif.MessageID != chatMessages[0].MessageID {
		t.Errorf("Received incorrect message ID for ACK. Expected %s, got %s", chatMessages[0].MessageID, notif.MessageID)
	}

	if !network.Nodes()[1].isDuplicate(&pb.Message{MessageID: chatMessages[0].MessageID}) {
		t.Error("Message is not marked as duplicate on node 0 when it should be")
	}
}

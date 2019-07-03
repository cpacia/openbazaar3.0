package core

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
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

	network.Nodes()[1].sendAckMessage(network.Nodes()[0].Identity(), &pb.Message{MessageID: chatMessages[0].MessageID, MessageType: pb.Message_CHAT_MESSAGE})

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

func TestOpenBazaarNode_maybeParkMessage(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	defer node.DestroyNode()

	p, err := peer.IDB58Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}

	parked, err := node.maybeParkMessage(p, &pb.Message{
		MessageType: pb.Message_CHAT_MESSAGE,
	})
	if err != nil {
		t.Fatal(err)
	}

	if parked {
		t.Error("First message should not be parked")
	}

	var messages []models.ParkedMessage
	err = node.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("message_type = ?", models.PmtChat).Find(&messages).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 0 {
		t.Errorf("Incorrect number of parked messages returned. Expected %d, got %d", 0, len(messages))
	}

	parked, err = node.maybeParkMessage(p, &pb.Message{
		MessageType: pb.Message_CHAT_MESSAGE,
		Sequence: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !parked {
		t.Error("Second message should be parked")
	}

	err = node.repo.DB().View(func(tx *gorm.DB) error {
		return tx.Where("message_type = ?", models.PmtChat).Find(&messages).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 1 {
		t.Errorf("Incorrect number of parked messages returned. Expected %d, got %d", 1, len(messages))
	}
}

func Test_prepMessage(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	defer node.DestroyNode()

	p, err := peer.IDB58Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}
	var message *pb.Message
	err = node.repo.DB().Update(func(tx *gorm.DB)error {
		message, err = prepMessage(tx, p, pb.Message_CHAT_MESSAGE)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	if message.Sequence != 1 {
		t.Errorf("Incorrect sequence. Expected %d, got %d", 1, message.Sequence)
	}

	err = node.repo.DB().Update(func(tx *gorm.DB)error {
		message, err = prepMessage(tx, p, pb.Message_CHAT_MESSAGE)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	if message.Sequence != 2 {
		t.Errorf("Incorrect sequence. Expected %d, got %d", 2, message.Sequence)
	}
}
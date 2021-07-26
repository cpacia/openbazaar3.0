package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"testing"
	"time"
)

func Test_SendAndReceiveAcks(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	sub, err := network.Nodes()[1].SubscribeEvent(&events.ChatMessage{})
	if err != nil {
		t.Fatal(err)
	}

	if err := network.Nodes()[0].SendChatMessage(network.nodes[1].Identity(), "hola", "", nil); err != nil {
		t.Fatal(err)
	}

	<-sub.Out()

	var chatMessages []models.ChatMessage
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Model(&models.ChatMessage{}).Find(&chatMessages).Error
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
	err = network.Nodes()[1].repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Model(&models.IncomingMessage{}).Find(&incomingMessages).Error
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

func TestOpenBazaarNode_syncMessages(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	listing := factory.NewPhysicalListing("tshirt")

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Cancelable order
	// We're going to disconnect the nodes, make the purchase, and then reconnect. This should cause node 1
	// to resend the order upon reconnection.
	network.Nodes()[0].networkService.Close()
	go network.Nodes()[1].syncMessages()
	if err := network.ipfsNet.UnlinkPeers(network.Nodes()[0].Identity(), network.Nodes()[1].Identity()); err != nil {
		t.Fatal(err)
	}
	err = network.Nodes()[1].SendChatMessage(network.Nodes()[0].Identity(), "message1", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = network.Nodes()[1].SendChatMessage(network.Nodes()[0].Identity(), "message2", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = network.Nodes()[1].SendChatMessage(network.Nodes()[0].Identity(), "message3", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	sub, err := network.Nodes()[0].eventBus.Subscribe(&events.ChatMessage{})
	if err != nil {
		t.Fatal(err)
	}


	// Reconnecting nodes should trigger node 1 to send the messages to node 0 again.
	time.Sleep(1)
	network.Nodes()[0].networkService = net.NewNetworkService(network.Nodes()[0].ipfsNode.PeerHost, net.NewBanManager(nil), true)
	network.Nodes()[0].registerHandlers()

	if _, err := network.ipfsNet.LinkPeers(network.Nodes()[0].Identity(), network.Nodes()[1].Identity()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-sub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	messages, err := network.Nodes()[1].GetChatMessagesByPeer(network.Nodes()[0].Identity(), -1, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(messages) != 3 {
		t.Errorf("Incorrect number of messages. Expected %d, got %d", 3, len(messages))
	}
}

func TestOpenBazaarNode_PublishToFollowers(t *testing.T) {
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer mocknet.TearDown()

	// Start node - follower tracker
	mocknet.Nodes()[0].followerTracker.Start()

	storeSub, err := mocknet.Nodes()[1].SubscribeEvent(&events.MessageStore{})
	if err != nil {
		t.Fatal(err)
	}

	followSub, err := mocknet.Nodes()[0].SubscribeEvent(&events.TrackerFollow{})
	if err != nil {
		t.Fatal(err)
	}

	// Set profile node 0
	done1 := make(chan struct{})
	if err := mocknet.Nodes()[0].SetProfile(&models.Profile{Name: "Peter Griffin"}, done1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done1:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Node 1 send follow
	done2 := make(chan struct{})
	if err := mocknet.Nodes()[1].FollowNode(mocknet.Nodes()[0].Identity(), done2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done2:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	select {
	case <-followSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Run the follower tracker to load node 1 as a follower in node 0.
	mocknet.Nodes()[0].followerTracker.tryConnectFollowers()

	// Set profile again with a new publish.
	done3 := make(chan struct{})
	if err := mocknet.Nodes()[0].SetProfile(&models.Profile{Name: "Peter Griffin2"}, done3); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done3:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	// Make sure 1 node is pinning the correct cids.
	select {
	case <-storeSub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	graph, err := mocknet.Nodes()[0].fetchGraph(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, cid := range graph {
		has, err := mocknet.Nodes()[1].ipfsNode.Blockstore.Has(cid)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Error("Missing cid")
		}
	}
}

func TestOpenBazaarNode_republish(t *testing.T) {
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer mocknet.TearDown()

	sub, err := mocknet.Nodes()[0].SubscribeEvent(&events.PublishFinished{})
	if err != nil {
		t.Fatal(err)
	}

	mocknet.Nodes()[0].publishChan <- pubCloser{
		nil,
	}

	select {
	case <-sub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
}

package net

import (
	"context"
	"github.com/cpacia/openbazaar3.0/net/pb"
	peer "github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"testing"
)

func TestNetworkService(t *testing.T) {
	mocknet, err := mocknet.FullMeshLinked(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	service1 := NewNetworkService(mocknet.Hosts()[0], NewBanManager(nil), true)
	service2 := NewNetworkService(mocknet.Hosts()[1], NewBanManager(nil), true)

	ms, err := service1.messageSenderForPeer(context.Background(), mocknet.Hosts()[1].ID())
	if err != nil {
		t.Fatal(err)
	}

	ch := make(chan struct{})
	service2.RegisterHandler(pb.Message_ACK, func(p peer.ID, msg *pb.Message) error {
		ch <- struct{}{}
		return nil
	})

	if err := ms.sendMessage(context.Background(), &pb.Message{}); err != nil {
		t.Error(err)
	}

	<-ch
}

package test

import (
	"testing"
	"time"
)

func TestChat(t *testing.T) {
	network, err := NewTestNetwork(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	done := make(chan struct{})
	if err := network.Nodes[0].SendChatMessage(network.Nodes[1].Identity(), "Hello there!", "", done); err != nil {
		t.Error(err)
	}
	<-done
	time.Sleep(time.Second*1)
}

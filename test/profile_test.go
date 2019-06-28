package test

import (
	"github.com/cpacia/openbazaar3.0/models"
	"testing"
)

func TestSetAndGetProfile(t *testing.T) {
	network, err := NewTestNetwork(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	for _, node := range network.Nodes {
		node.Start()
	}

	name := "Chris"

	done := make(chan struct{})
	pro := &models.Profile{
		Name: name,
	}
	if err := network.Nodes[0].SetProfile(pro, done); err != nil {
		t.Fatal(err)
	}
	<-done

	profile, err := network.Nodes[1].GetProfile(network.Nodes[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != name {
		t.Fatalf("Invalid name. Expected %s, got %s", name, profile.Name)
	}

	profile, err = network.Nodes[1].GetProfile(network.Nodes[0].Identity(), true)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != name {
		t.Fatalf("Invalid name. Expected %s, got %s", name, profile.Name)
	}
}

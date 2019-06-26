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

	for _, node := range network.Nodes {
		node.Start()
	}

	name := "Chris"

	err = network.Nodes[0].SetProfile(&models.Profile{
		Name: name,
	})
	if err != nil {
		t.Fatal(err)
	}

	profile, err := network.Nodes[1].GetProfile(network.Nodes[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != name {
		t.Fatalf("Invalid name. Expected %s, got %s", name, profile.Name)
	}
}

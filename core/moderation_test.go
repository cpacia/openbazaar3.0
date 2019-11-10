package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/models"
	peer "github.com/libp2p/go-libp2p-peer"
	"testing"
	"time"
)

func TestOpenBazaarNode_SetAndRemoveSelfAsModerator(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	defer node.DestroyNode()

	if err := node.SetProfile(&models.Profile{Name: "Ron Paul"}, nil); err != nil {
		t.Fatal(err)
	}

	modInfo := &models.ModeratorInfo{
		Fee: models.ModeratorFee{
			FeeType:    models.PercentageFee,
			Percentage: 10,
		},
	}

	done := make(chan struct{})
	if err := node.SetSelfAsModerator(context.Background(), modInfo, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	done2 := make(chan struct{})
	if err := node.RemoveSelfAsModerator(context.Background(), done2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done2:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
}

func TestOpenBazaarNode_GetModerators(t *testing.T) {
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer mocknet.TearDown()

	done0 := make(chan struct{})
	originalProfile := &models.Profile{Name: "Ron Paul"}
	if err := mocknet.Nodes()[0].SetProfile(originalProfile, done0); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done0:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	modInfo := &models.ModeratorInfo{
		Fee: models.ModeratorFee{
			FeeType:    models.PercentageFee,
			Percentage: 10,
		},
	}

	done := make(chan struct{})
	if err := mocknet.Nodes()[0].SetSelfAsModerator(context.Background(), modInfo, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	mods := mocknet.Nodes()[1].GetModerators(context.Background())

	if len(mods) != 1 {
		t.Errorf("Returned incorrect number of moderators. Expected %d, got %d", 1, len(mods))
	}

	if mods[0].Pretty() != mocknet.Nodes()[0].Identity().Pretty() {
		t.Errorf("Returned incorrect peer ID. Expected %s, got %s", mocknet.Nodes()[0].Identity().Pretty(), mods[0].Pretty())
	}

	ch := mocknet.Nodes()[1].GetModeratorsAsync(context.Background())

	mods = []peer.ID{}
	for mod := range ch {
		mods = append(mods, mod)
	}

	if len(mods) != 1 {
		t.Errorf("Returned incorrect number of moderators. Expected %d, got %d", 1, len(mods))
	}

	if mods[0].Pretty() != mocknet.Nodes()[0].Identity().Pretty() {
		t.Errorf("Returned incorrect peer ID. Expected %s, got %s", mocknet.Nodes()[0].Identity().Pretty(), mods[0].Pretty())
	}

	profile, err := mocknet.Nodes()[1].GetProfile(context.Background(), mods[0], false)
	if err != nil {
		t.Fatal(err)
	}

	if profile.Name != originalProfile.Name {
		t.Errorf("Returned incorrect profile name. Expected %s, got %s", originalProfile.Name, profile.Name)
	}

	if profile.ModeratorInfo == nil {
		t.Error("Profile moderator info is nil")
	}

	if !profile.Moderator {
		t.Error("Profile does not have moderator bool set")
	}

	if profile.ModeratorInfo.Fee.Percentage != modInfo.Fee.Percentage {
		t.Errorf("Returned incorrect moderator percentage. Expected %f, got %f", modInfo.Fee.Percentage, profile.ModeratorInfo.Fee.Percentage)
	}
}

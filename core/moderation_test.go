package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/models"
	"testing"
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
			FeeType: models.PercentageFee,
			Percentage: 10,
		},
	}

	done := make(chan struct{})
	if err := node.SetSelfAsModerator(context.Background(), modInfo, done); err != nil {
		t.Fatal(err)
	}
	<-done

	done2 := make(chan struct{})
	if err := node.RemoveSelfAsModerator(context.Background(), done2); err != nil {
		t.Fatal(err)
	}
	<-done2
}
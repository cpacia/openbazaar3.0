package core

import (
	iwallet "github.com/cpacia/wallet-interface"
	"testing"
)

func TestOpenBazaarNode_RequestAddress(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer network.TearDown()

	address, err := network.Nodes()[0].RequestAddress(network.Nodes()[1].Identity(), iwallet.CtTestnetMock)
	if err != nil {
		t.Fatal(err)
	}

	if address.CoinType() != iwallet.CtTestnetMock {
		t.Errorf("Incorrect cointype expected TMCK got %s", address.CoinType().CurrencyCode())
	}
	if len(address.String()) != 40 {
		t.Errorf("Expected address length of 20 got %d", len(address.String()))
	}

	_, err = network.Nodes()[0].RequestAddress(network.Nodes()[1].Identity(), iwallet.CtBitcoin)
	if err == nil {
		t.Error("Expected request for unknown cointype to error")
	}
}

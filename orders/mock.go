package orders

import (
	"context"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-ipfs/core"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

func newMockOrderProcessor() (*OrderProcessor, error) {
	ctx := context.Background()

	mn := mocknet.New(ctx)
	ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
		Online: true,
		Host:   coremock.MockHostOption(mn),
	})
	if err != nil {
		return nil, err
	}

	r, err := repo.MockRepo()
	if err != nil {
		return nil, err
	}

	banManager := net.NewBanManager(nil)
	service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

	messenger := net.NewMessenger(service, r.DB())

	repo, err := repo.MockRepo()
	if err != nil {
		return nil, err
	}

	wal := wallet.NewMockWallet()
	mw := make(wallet.Multiwallet)
	mw[iwallet.CtTestnetMock] = wal

	return NewOrderProcessor(ipfsNode.Identity, repo.DB(), messenger, mw), nil
}

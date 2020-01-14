package orders

import (
	"context"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-ipfs/core"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"path"
)

func newMockOrderProcessor() (*OrderProcessor, func(), error) {
	r, err := repo.MockRepo()
	if err != nil {
		return nil, nil, err
	}

	ipfsRepo, err := fsrepo.Open(path.Join(r.DataDir(), "ipfs"))
	if err != nil {
		return nil, nil, err
	}

	ipfsConfig, err := ipfsRepo.Config()
	if err != nil {
		return nil, nil, err
	}

	ipfsConfig.Bootstrap = nil

	var dbIdentityKey models.Key
	err = r.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	ipfsConfig.Identity, err = repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()

	mn := mocknet.New(ctx)

	ipfsNode, err := core.NewNode(ctx, &core.BuildCfg{
		Online: true,
		Repo:   ipfsRepo,
		Host:   coremock.MockHostOption(mn),
	})
	if err != nil {
		return nil, nil, err
	}

	banManager := net.NewBanManager(nil)
	service := net.NewNetworkService(ipfsNode.PeerHost, banManager, true)

	messenger, err := net.NewMessenger(&net.MessengerConfig{
		Privkey: ipfsNode.PrivateKey,
		Service: service,
		DB:      r.DB(),
		Context: ipfsNode.Context(),
	})
	if err != nil {
		return nil, nil, err
	}

	mw := multiwallet.Multiwallet{
		iwallet.CtMock: wallet.NewMockWallet(),
	}

	erp, err := wallet.NewMockExchangeRates()
	if err != nil {
		return nil, nil, err
	}

	return NewOrderProcessor(&Config{
			Identity:             ipfsNode.Identity,
			IdentityPrivateKey:   ipfsNode.PrivateKey,
			Db:                   r.DB(),
			Messenger:            messenger,
			Multiwallet:          mw,
			ExchangeRateProvider: erp,
			EventBus:             events.NewBus(),
		}), func() {
			ipfsNode.Close()
			r.DestroyRepo()
		}, nil
}

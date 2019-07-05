package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	"github.com/cpacia/openbazaar3.0/repo"
	bitswap "github.com/ipfs/go-bitswap/network"
	"os"
	"path"
	"testing"
)

func TestNewNode(t *testing.T) {
	dataDir := path.Join(os.TempDir(), "openbazaar-test", "TestNewNode")

	cfg := repo.Config{
		DataDir:       dataDir,
		Testnet:       true,
		IPNSQuorum:    3,
		BoostrapAddrs: []string{},
	}

	node, err := NewNode(context.Background(), &cfg)
	if err != nil {
		t.Fatal(err)
	}

	defer node.DestroyNode()

	if bitswap.ProtocolBitswap != net.ProtocolBitswapMainnetTwo {
		t.Error("Failed to set correct bitswap protocol")
	}

	// Load our identity key from the db and set it in the config.
	var dbIdentityKey models.Key
	err = node.repo.DB().View(func(tx database.Tx) error {
		return tx.DB().Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	id, err := repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		t.Fatal(err)
	}
	if node.ipfsNode.Identity.Pretty() != id.PeerID {
		t.Error("Incorrect identity instantiated")
	}
}

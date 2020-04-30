package core

import (
	"context"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/repo"
	"math/rand"
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
		SwarmAddrs:    []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", rand.Intn(65535))},
	}

	node, err := NewNode(context.Background(), &cfg)
	if err != nil {
		t.Fatal(err)
	}

	defer node.DestroyNode()

	// Load our identity key from the db and set it in the config.
	var dbIdentityKey models.Key
	err = node.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("name = ?", "identity").First(&dbIdentityKey).Error
	})

	id, err := repo.IdentityFromKey(dbIdentityKey.Value)
	if err != nil {
		t.Fatal(err)
	}
	if node.ipfsNode.Identity.Pretty() != id.PeerID {
		t.Error("Incorrect identity instantiated")
	}
}

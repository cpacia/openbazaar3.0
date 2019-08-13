package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/models"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/op/go-logging"
	"github.com/tyler-smith/go-bip39"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

var log = logging.MustGetLogger("REPO")

func init() {
	// Install IPFS database plugins. This is guarded by a sync.Once.
	installDatabasePlugins()
}

// Repo is a representation of an OpenBazaar data directory.
// In this we store:
// - IPFS node data
// - The openbazaar.conf file
// - The node's data root directory
// - The OpenBazaar database
// - A wallet directory which holds wallet plugin data
type Repo struct {
	db      database.Database
	dataDir string
}

// NewRepo returns a new Repo for the given data directory. It will
// be initialized if it is not already.
func NewRepo(dataDir string) (*Repo, error) {
	return newRepo(dataDir, "", false)
}

// NewRepoWithCustomMnemonicSeed behaves the same as NewRepo but allows
// the caller to pass in a custom mnemonic seed. This is usuful for
// restoring a node from seed.
func NewRepoWithCustomMnemonicSeed(dataDir, mnemonic string) (*Repo, error) {
	return newRepo(dataDir, mnemonic, false)
}

// DB returns the database implementation.
func (r *Repo) DB() database.Database {
	return r.db
}

// DataDir returns the data directory associated with this repo.
func (r *Repo) DataDir() string {
	return r.dataDir
}

// Close will close the repo and associated databases.
func (r *Repo) Close() {
	r.db.Close()
}

// DestroyRepo deletes the entire directory. Do NOT use this unless you are
// positive you want to wipe all data.
func (r *Repo) DestroyRepo() error {
	return os.RemoveAll(r.dataDir)
}

func newRepo(dataDir, mnemonicSeed string, inMemoryDB bool) (*Repo, error) {
	var (
		dbIdentity, dbSeed, dbMnemonic *models.Key
		err                            error
	)
	if !fsrepo.IsInitialized(dataDir) {
		if err := checkWriteable(dataDir); err != nil {
			fmt.Println(err)
			return nil, err
		}
		if mnemonicSeed == "" {
			mnemonicSeed, err = createMnemonic(bip39.NewEntropy, bip39.NewMnemonic)
			if err != nil {
				return nil, err
			}
		}
		seed := bip39.NewSeed(mnemonicSeed, "Secret Passphrase")
		identityKey, err := IdentityKeyFromSeed(seed, 0)
		if err != nil {
			return nil, err
		}

		identity, err := IdentityFromKey(identityKey)
		if err != nil {
			return nil, err
		}
		conf := mustDefaultConfig()
		conf.Identity = identity
		if err := fsrepo.Init(dataDir, conf); err != nil {
			return nil, err
		}

		if err := initializeIpnsKeyspace(dataDir, identityKey); err != nil {
			return nil, err
		}
		dbIdentity = &models.Key{
			Name:  "identity",
			Value: identityKey,
		}
		dbSeed = &models.Key{
			Name:  "seed",
			Value: seed,
		}
		dbMnemonic = &models.Key{
			Name:  "mnemonic",
			Value: []byte(mnemonicSeed),
		}
		if err := cleanIdentityFromConfig(dataDir); err != nil {
			return nil, err
		}
	}

	var db database.Database
	if inMemoryDB {
		db, err = ffsqlite.NewFFMemoryDB(dataDir)
		if err != nil {
			return nil, err
		}
	} else {
		db, err = ffsqlite.NewFFSqliteDB(dataDir)
		if err != nil {
			return nil, err
		}
	}

	if err := autoMigrateDatabase(db); err != nil {
		return nil, err
	}

	err = db.Update(func(tx database.Tx) error {
		if dbIdentity != nil {
			if err := tx.Save(&dbIdentity); err != nil {
				return err
			}
		}
		if dbSeed != nil {
			if err := tx.Save(&dbSeed); err != nil {
				return err
			}
		}
		if dbMnemonic != nil {
			if err := tx.Save(&dbMnemonic); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := CheckAndSetUlimit(); err != nil {
		return nil, err
	}

	return &Repo{
		dataDir: dataDir,
		db:      db,
	}, nil
}

func checkWriteable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// Directory exists, make sure we can write to it
		testfile := path.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// Directory does not exist, check that we can create it
		return os.MkdirAll(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

func createMnemonic(newEntropy func(int) ([]byte, error), newMnemonic func([]byte) (string, error)) (string, error) {
	entropy, err := newEntropy(128)
	if err != nil {
		return "", err
	}
	mnemonic, err := newMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func initializeIpnsKeyspace(repoRoot string, privKeyBytes []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

	cfg, err := r.Config()
	if err != nil {
		return err
	}
	identity, err := IdentityFromKey(privKeyBytes)
	if err != nil {
		return err
	}

	cfg.Identity = identity

	nd, err := core.NewNode(ctx, &core.BuildCfg{Repo: r})
	if err != nil {
		return err
	}
	defer nd.Close()

	return namesys.InitializeKeyspace(ctx, nd.Namesys, nd.Pinning, nd.PrivateKey)
}

func mustDefaultConfig() *config.Config {
	bootstrapPeers, err := config.ParseBootstrapPeers([]string{}) // TODO:
	if err != nil {
		// BootstrapAddressesDefault are local and should never panic
		panic(err)
	}

	conf, err := config.Init(&dummyWriter{}, 4096)
	if err != nil {
		panic(err)
	}
	conf.Ipns.RecordLifetime = "168h"
	conf.Ipns.RepublishPeriod = "12h"
	conf.Discovery.MDNS.Enabled = false
	conf.Addresses = config.Addresses{
		Swarm: []string{
			"/ip4/0.0.0.0/tcp/4001",
			"/ip6/::/tcp/4001",
			"/ip4/0.0.0.0/tcp/9005/ws",
			"/ip6/::/tcp/9005/ws",
		},
		API:     []string{""},
		Gateway: []string{"/ip4/127.0.0.1/tcp/4002"},
	}
	conf.Bootstrap = config.BootstrapPeerStrings(bootstrapPeers)

	return conf
}

type dummyWriter struct{}

func (d *dummyWriter) Write(p []byte) (n int, err error) { return 0, nil }

var pluginOnce sync.Once

// installDatabasePlugins installs the default database plugins
// used by openbazaar-go. This function is guarded by a sync.Once
// so it isn't accidentally called more than once.
func installDatabasePlugins() {
	pluginOnce.Do(func() {
		loader, err := loader.NewPluginLoader("")
		if err != nil {
			panic(err)
		}
		err = loader.Initialize()
		if err != nil {
			panic(err)
		}

		err = loader.Inject()
		if err != nil {
			panic(err)
		}
	})
}

// The IPFS config file holds the private key to the node. First we aren't
// even using this key as we prefer to use one derived from a mnemonic, but
// second we don't want it sitting in the config file anyway. So this function
// removes the identity object from the config. The identity object will be
// added back into the config with the correct key/identity by the OpenBazaarNode
// builder.
func cleanIdentityFromConfig(dataDir string) error {
	configPath := path.Join(dataDir, "config")
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfgIface interface{}
	if err := json.Unmarshal(configFile, &cfgIface); err != nil {
		return err
	}
	cfg, ok := cfgIface.(map[string]interface{})
	if !ok {
		return errors.New("invalid config file")
	}
	delete(cfg, "Identity")
	out, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, out, os.ModePerm)
}

func autoMigrateDatabase(db database.Database) error {
	dbModels := []interface{}{
		&models.Key{},
		&models.CachedIPNSEntry{},
		&models.OutgoingMessage{},
		&models.IncomingMessage{},
		&models.ChatMessage{},
		&models.NotificationRecord{},
		&models.FollowerStat{},
		&models.FollowSequence{},
		&models.Coupon{},
		&models.Event{},
		&models.Order{},
	}

	return db.Update(func(tx database.Tx) error {
		for _, m := range dbModels {
			if err := tx.Migrate(m); err != nil {
				return err
			}
		}
		return nil
	})
}

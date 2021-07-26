package repo

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/version"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/fuse/ipns"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/op/go-logging"
	"github.com/tyler-smith/go-bip39"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
)

const (
	// defaultRepoVersion is the current repo version used for migrations.
	defaultRepoVersion = 0

	// versionFileName is the name of the version file.
	versionFileName = "version"

	// defaultMispaymentBuffer is the default buffer to use when calculating a
	// mispayment.
	defaultMispaymentBuffer = 1.0
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
	if err := r.db.Close(); err != nil {
		return err
	}
	return os.RemoveAll(r.dataDir)
}

// writeVersion writes the version number to file.
func (r *Repo) writeVersion(version int) error {
	versionStr := strconv.Itoa(version)
	return ioutil.WriteFile(path.Join(r.dataDir, versionFileName), []byte(versionStr), os.ModePerm)
}

func newRepo(dataDir, mnemonicSeed string, inMemoryDB bool) (*Repo, error) {
	var (
		dbIdentity, dbEscrowKey, dbRatingKey, dbBip44Key, dbMnemonic, torKey *models.Key
		err                                                                  error
		isNew                                                                bool
	)
	ipfsDir := path.Join(dataDir, "ipfs")
	if !fsrepo.IsInitialized(ipfsDir) {
		if err := checkWriteable(ipfsDir); err != nil {
			return nil, err
		}
		if mnemonicSeed == "" {
			mnemonicSeed, err = createMnemonic(bip39.NewEntropy, bip39.NewMnemonic)
			if err != nil {
				return nil, err
			}
		}

		identitySeed := bip39.NewSeed(mnemonicSeed, "Secret Passphrase")
		identityKey, err := IdentityKeyFromSeed(identitySeed, 0)
		if err != nil {
			return nil, err
		}

		identity, err := IdentityFromKey(identityKey)
		if err != nil {
			return nil, err
		}
		conf := mustDefaultConfig()
		conf.Identity = identity
		if err := fsrepo.Init(ipfsDir, conf); err != nil {
			return nil, err
		}

		if err := initializeIpnsKeyspace(ipfsDir, identityKey); err != nil {
			return nil, err
		}

		hdSeed := bip39.NewSeed(mnemonicSeed, "")
		escrowKey, ratingKey, bip44Key, err := createHDKeys(hdSeed)
		if err != nil {
			return nil, err
		}

		_, torPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		dbIdentity = &models.Key{
			Name:  "identity",
			Value: identityKey,
		}
		dbEscrowKey = &models.Key{
			Name:  "escrow",
			Value: escrowKey.Serialize(),
		}
		dbRatingKey = &models.Key{
			Name:  "ratings",
			Value: ratingKey.Serialize(),
		}
		dbBip44Key = &models.Key{
			Name:  "bip44",
			Value: []byte(bip44Key.String()),
		}
		dbMnemonic = &models.Key{
			Name:  "mnemonic",
			Value: []byte(mnemonicSeed),
		}
		torKey = &models.Key{
			Name:  "tor",
			Value: torPriv.Seed(),
		}
		if err := cleanIdentityFromConfig(ipfsDir); err != nil {
			return nil, err
		}
		isNew = true
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
		if dbEscrowKey != nil {
			if err := tx.Save(&dbEscrowKey); err != nil {
				return err
			}
		}
		if dbRatingKey != nil {
			if err := tx.Save(&dbRatingKey); err != nil {
				return err
			}
		}
		if dbBip44Key != nil {
			if err := tx.Save(&dbBip44Key); err != nil {
				return err
			}
		}
		if dbMnemonic != nil {
			if err := tx.Save(&dbMnemonic); err != nil {
				return err
			}
		}
		if torKey != nil {
			if err := tx.Save(&torKey); err != nil {
				return err
			}
		}
		if isNew {
			err := tx.Save(&models.UserPreferences{
				AutoConfirm:       true,
				MisPaymentBuffer:  defaultMispaymentBuffer,
				ShowNsfw:          true,
				ShowNotifications: true,
			})
			if err != nil {
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

	r := &Repo{
		dataDir: dataDir,
		db:      db,
	}
	if isNew {
		if err := r.writeVersion(defaultRepoVersion); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Repo) WriteUserAgent(comment string) error {
	return ioutil.WriteFile(path.Join(r.db.PublicDataPath(), "user_agent"), []byte(fmt.Sprintf("%s%s", version.UserAgent(), comment)), os.ModePerm)
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

func createHDKeys(seed []byte) (escrowKey, ratingKey *btcec.PrivateKey, bip44Key *hdkeychain.ExtendedKey, err error) {
	masterPrivKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, nil, nil, err
	}

	twoZeroNine, err := masterPrivKey.Child(hdkeychain.HardenedKeyStart + 209)
	if err != nil {
		return nil, nil, nil, err
	}

	bip44Key, err = masterPrivKey.Child(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, nil, nil, err
	}

	escrowHDKey, err := twoZeroNine.Child(0)
	if err != nil {
		return nil, nil, nil, err
	}

	ratingHDKey, err := twoZeroNine.Child(1)
	if err != nil {
		return nil, nil, nil, err
	}

	escrowKey, err = escrowHDKey.ECPrivKey()
	if err != nil {
		return nil, nil, nil, err
	}

	ratingKey, err = ratingHDKey.ECPrivKey()
	if err != nil {
		return nil, nil, nil, err
	}

	return escrowKey, ratingKey, bip44Key, nil
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

	return ipns.InitializeKeyspace(nd, nd.PrivateKey)
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
	conf.Ipns.RecordLifetime = "720h"
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
	conf.Swarm.EnableAutoRelay = true

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
		&models.TransactionMetadata{},
		&models.UserPreferences{},
		&models.StoreAndForwardServers{},
		&models.Case{},
		&models.Channel{},
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

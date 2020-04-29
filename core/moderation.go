package core

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"strings"
)

const (
	// moderatorTopic is the DHT key at which moderator "providers" are stored.
	moderatorTopic = "openbazaar:moderators"

	// moderatorCid is the cid path of the provider block.
	moderatorCid = "QmV9mSiAvEMvv6JyVYFaojPb4Se3XSpb4tW35AcjGfVqxb"

	// maxModerators is the maximum number of moderators to return in a single query.
	maxModerators = 1000
)

// SetSelfAsModerator sets this node as a node that is offering moderation services.
// It will update the profile with the moderation info, set itsef as a moderator
// in the DHT so it can be discovered by other peers, and publish.
func (n *OpenBazaarNode) SetSelfAsModerator(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error {
	if (int(modInfo.Fee.FeeType) == 0 || int(modInfo.Fee.FeeType) == 2) && modInfo.Fee.FixedFee == nil {
		maybeCloseDone(done)
		return errors.New("fixed fee must be set when using a fixed fee type")
	}

	err := n.repo.DB().Update(func(tx database.Tx) error {
		var (
			prefs      models.UserPreferences
			currencies []string
		)
		err := tx.Read().First(&prefs).Error
		if err == nil {
			currencies, err = prefs.PreferredCurrencies()
			if err != nil {
				return err
			}
			for _, cc := range currencies {
				modInfo.AcceptedCurrencies = append(modInfo.AcceptedCurrencies, normalizeCurrencyCode(cc))
			}
		}

		if len(currencies) == 0 {
			for ct := range n.multiwallet {
				currencies = append(currencies, ct.CurrencyCode())
			}
			for _, cc := range currencies {
				modInfo.AcceptedCurrencies = append(modInfo.AcceptedCurrencies, normalizeCurrencyCode(cc))
			}
		}

		profile, err := tx.GetProfile()
		if err != nil {
			return err
		}
		profile.ModeratorInfo = modInfo
		profile.Moderator = true

		if err := tx.SetProfile(profile); err != nil {
			return err
		}

		api, err := coreapi.NewCoreAPI(n.ipfsNode)
		if err != nil {
			return err
		}
		// This sets us as a "provider" in the DHT for the moderator key.
		// Other peers can find us by doing a DHT GetProviders query for
		// the same key.
		_, err = api.Block().Put(ctx, strings.NewReader(moderatorTopic))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
		return err
	}
	n.Publish(done)
	return nil
}

// RemoveSelfAsModerator removes this node as a moderator in the DHT and updates
// the profile and publishes.
func (n *OpenBazaarNode) RemoveSelfAsModerator(ctx context.Context, done chan<- struct{}) error {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		profile, err := tx.GetProfile()
		if err != nil {
			return err
		}
		profile.Moderator = true

		if err := tx.SetProfile(profile); err != nil {
			return err
		}

		api, err := coreapi.NewCoreAPI(n.ipfsNode)
		if err != nil {
			return err
		}
		return api.Block().Rm(ctx, path.New(moderatorCid))
	})
	if err != nil {
		maybeCloseDone(done)
		return err
	}
	n.Publish(done)
	return nil
}

// GetModerators returns a slice of moderators found on the network.
func (n *OpenBazaarNode) GetModerators(ctx context.Context) []peer.ID {
	var mods []peer.ID
	for mod := range n.GetModeratorsAsync(ctx) {
		mods = append(mods, mod)
	}
	return mods
}

// GetModeratorsAsync returns a chan over which new moderator IDs are pushed.
func (n *OpenBazaarNode) GetModeratorsAsync(ctx context.Context) <-chan peer.ID {
	ch := make(chan peer.ID)

	go func() {
		c, err := cid.Decode(moderatorCid)
		if err != nil {
			log.Errorf("Error decoding moderator cid: %s", err)
			close(ch)
			return
		}
		provCh := n.ipfsNode.Routing.FindProvidersAsync(ctx, c, maxModerators)

		for prov := range provCh {
			ch <- prov.ID
		}
		close(ch)
	}()

	return ch
}

// SetModeratorsOnListings updates all listings with the new moderators and publishes
// the changes.
func (n *OpenBazaarNode) SetModeratorsOnListings(mods []peer.ID, done chan struct{}) error {
	modStrs := make([]string, 0, len(mods))
	for _, mod := range mods {
		modStrs = append(modStrs, mod.Pretty())
	}

	return n.UpdateAllListings(func(listing *pb.Listing) (bool, error) {
		listing.Moderators = modStrs
		return true, nil
	}, done)
}

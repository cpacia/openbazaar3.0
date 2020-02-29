package core

import (
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"os"
)

// GetPreferences returns the saved preferences for this node.
func (n *OpenBazaarNode) GetPreferences() (*models.UserPreferences, error) {
	var prefs models.UserPreferences
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().First(&prefs).Error
	})
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

// SavePreferences saves the preferences in the database and updates the moderators
// on the store of they are different.
func (n *OpenBazaarNode) SavePreferences(prefs *models.UserPreferences, done chan struct{}) error {
	var modsChanged bool
	err := n.repo.DB().Update(func(tx database.Tx) error {
		var (
			currentPrefs  models.UserPreferences
			currentModMap = make(map[peer.ID]bool)
			newModMap     = make(map[peer.ID]bool)
		)
		err := tx.Read().First(&currentPrefs).Error
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}

		currentMods, err := currentPrefs.StoreModerators()
		if err != nil {
			return err
		}
		for _, mod := range currentMods {
			currentModMap[mod] = true
		}

		mods, err := prefs.StoreModerators()
		if err != nil {
			return fmt.Errorf("%w: invalid moderator ID", coreiface.ErrBadRequest)
		}
		for _, mod := range mods {
			newModMap[mod] = true
			if !currentModMap[mod] {
				modsChanged = true
			}
		}
		for _, mod := range currentMods {
			if !newModMap[mod] {
				modsChanged = true
			}
		}

		// Validate blocked nodes
		_, err = prefs.BlockedNodes()
		if err != nil {
			return fmt.Errorf("%w: invalid block node ID", coreiface.ErrBadRequest)
		}

		currencies, err := prefs.PreferredCurrencies()
		if err != nil {
			return err
		}
		for _, cur := range currencies {
			_, err = n.multiwallet.WalletForCurrencyCode(cur)
			if err != nil {
				return fmt.Errorf("%w: no wallet for currency %s", coreiface.ErrBadRequest, cur)
			}
		}
		prefs.ID = 1
		if err := tx.Save(prefs); err != nil {
			return err
		}
		_, err = tx.GetListingIndex()
		if modsChanged && !os.IsNotExist(err) {
			modStrs := make([]string, 0, len(mods))
			for _, mod := range mods {
				modStrs = append(modStrs, mod.Pretty())
			}
			_, err := n.updateAllListings(tx, func(l *pb.Listing) (bool, error) {
				l.Moderators = modStrs
				return true, nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
		return err
	}
	if modsChanged {
		n.Publish(done)
	}
	return nil
}

package core

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"os"
	"strconv"
	"time"
)

// SetProfile sets the public profile for the node and publishes to IPNS.
//
// The publishing is done in a separate goroutine so this function will
// return as soon as the profile is saved to disk. The optional done
// chan will be closed when publishing is complete.
func (n *OpenBazaarNode) SetProfile(profile *models.Profile, done chan<- struct{}) error {
	profile.EscrowPublicKey = hex.EncodeToString(n.escrowMasterKey.PubKey().SerializeCompressed())
	profile.PeerID = n.Identity().Pretty()
	profile.LastModified = time.Now()
	profile.StoreAndForwardServers = n.storeAndForwardServers

	if err := validateProfile(profile); err != nil {
		if done != nil {
			close(done)
		}
		return fmt.Errorf("%w: %s", coreiface.ErrBadRequest, err)
	}

	err := n.repo.DB().Update(func(tx database.Tx) error {
		var prefs models.UserPreferences
		if err := tx.Read().First(&prefs).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}

		if profile.Moderator && len(profile.ModeratorInfo.AcceptedCurrencies) == 0 {
			currencies, err := prefs.PreferredCurrencies()
			if err != nil {
				return err
			}
			profile.ModeratorInfo.AcceptedCurrencies = currencies
		}

		if err := n.updateProfileStats(tx, profile); err != nil {
			return err
		}
		if err := tx.SetProfile(profile); err != nil {
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

// GetMyProfile returns the profile for this node.
func (n *OpenBazaarNode) GetMyProfile() (*models.Profile, error) {
	var (
		profile *models.Profile
		err     error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		profile, err = tx.GetProfile()
		if err != nil {
			return fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
		}
		return nil
	})
	return profile, err
}

// GetProfile returns the profile of the node with the given peer ID.
// If useCache is set it will return a profile from the local cache
// (if it has one) if profile is not found on the network.
func (n *OpenBazaarNode) GetProfile(ctx context.Context, peerID peer.ID, useCache bool) (*models.Profile, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	profileBytes, err := n.cat(ctx, path.Join(pth, ffsqlite.ProfileFile))
	if err != nil {
		return nil, err
	}
	profile := new(models.Profile)
	if err := json.Unmarshal(profileBytes, profile); err != nil {
		return nil, err
	}
	if err := validateProfile(profile); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	if len(profile.StoreAndForwardServers) > 0 {
		err := n.repo.DB().Update(func(tx database.Tx) error {
			pi := models.StoreAndForwardServers{
				PeerID:      peerID.Pretty(),
				LastUpdated: time.Now(),
			}
			if err := pi.PutServers(profile.StoreAndForwardServers); err != nil {
				return err
			}
			return tx.Save(&pi)
		})
		if err != nil {
			return nil, err
		}
	}
	return profile, nil
}

// updateAndSaveProfile loads the profile from disk, updates
// the profile stats, then saves it back to disk.
func (n *OpenBazaarNode) updateAndSaveProfile(tx database.Tx) error {
	profile, err := tx.GetProfile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if profile == nil {
		return nil
	}
	if err := n.updateProfileStats(tx, profile); err != nil {
		return err
	}
	return tx.SetProfile(profile)
}

// updateProfileStats updates all stats on the passed in profile
func (n *OpenBazaarNode) updateProfileStats(tx database.Tx, profile *models.Profile) error {
	followers, err := tx.GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	following, err := tx.GetFollowing()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	listings, err := tx.GetListingIndex()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	profile.Stats = &models.ProfileStats{
		FollowerCount:  uint32(followers.Count()),
		FollowingCount: uint32(following.Count()),
		ListingCount:   uint32(listings.Count()),
	}

	return nil
}

// updateSNFServers will update the profile's store and forward servers
// if they have changed.
func (n *OpenBazaarNode) updateSNFServers() error {
	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i, v := range a {
			if v != b[i] {
				return false
			}
		}
		return true
	}
	updated := false
	err := n.repo.DB().Update(func(tx database.Tx) error {
		profile, err := tx.GetProfile()
		if err != nil {
			return err
		}
		if !equal(profile.StoreAndForwardServers, n.storeAndForwardServers) {
			profile.StoreAndForwardServers = n.storeAndForwardServers

			if err := n.updateProfileStats(tx, profile); err != nil {
				return err
			}
			if err := tx.SetProfile(profile); err != nil {
				return err
			}

			updated = true
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && updated {
		n.Publish(nil)
	}
	return nil
}

// validateProfile checks each field to make sure they're formatted properly and/or
// within the desired limits.
func validateProfile(profile *models.Profile) error {
	if len(profile.Name) == 0 {
		return coreiface.ErrMissingField("name")
	}
	if len(profile.Name) > WordMaxCharacters {
		return coreiface.ErrTooManyCharacters{"name", strconv.Itoa(WordMaxCharacters)}
	}
	if len(profile.Location) > WordMaxCharacters {
		return coreiface.ErrTooManyCharacters{"location", strconv.Itoa(WordMaxCharacters)}
	}
	if len(profile.About) > AboutMaxCharacters {
		return coreiface.ErrTooManyCharacters{"about", strconv.Itoa(AboutMaxCharacters)}
	}
	if len(profile.ShortDescription) > models.ShortDescriptionLength {
		return coreiface.ErrTooManyCharacters{"shortdescription", strconv.Itoa(models.ShortDescriptionLength)}
	}
	if profile.ContactInfo != nil {
		if len(profile.ContactInfo.Website) > URLMaxCharacters {
			return coreiface.ErrTooManyCharacters{"contactinfo.website", strconv.Itoa(URLMaxCharacters)}
		}
		if len(profile.ContactInfo.Email) > SentenceMaxCharacters {
			return coreiface.ErrTooManyCharacters{"contactinfo.email", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(profile.ContactInfo.PhoneNumber) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"contactinfo.phonenumber", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(profile.ContactInfo.Social) > MaxListItems {
			return coreiface.ErrTooManyItems{"contactinfo.social", strconv.Itoa(MaxListItems)}
		}
		for _, s := range profile.ContactInfo.Social {
			if len(s.Username) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"contactinfo.social.username", strconv.Itoa(WordMaxCharacters)}
			}
			if len(s.Type) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"contactinfo.social.type", strconv.Itoa(WordMaxCharacters)}
			}
			if len(s.Proof) > URLMaxCharacters {
				return coreiface.ErrTooManyCharacters{"contactinfo.social.proof", strconv.Itoa(URLMaxCharacters)}
			}
		}
	}
	if profile.Moderator && profile.ModeratorInfo == nil {
		return errors.New("moderatorinfo must be included if moderator boolean is set")
	}
	if profile.ModeratorInfo != nil {
		if (profile.ModeratorInfo.Fee.FeeType == models.FixedFee || profile.ModeratorInfo.Fee.FeeType == models.FixedPlusPercentageFee) && profile.ModeratorInfo.Fee.FixedFee == nil {
			return errors.New("moderator fee type must be set if using fixed fee or fixed plus percentage")
		}
		if len(profile.ModeratorInfo.Description) > AboutMaxCharacters {
			return coreiface.ErrTooManyCharacters{"moderatorinfo.description", strconv.Itoa(AboutMaxCharacters)}
		}
		if len(profile.ModeratorInfo.TermsAndConditions) > PolicyMaxCharacters {
			return coreiface.ErrTooManyCharacters{"moderatorinfo.termsandconditions", strconv.Itoa(PolicyMaxCharacters)}
		}
		if len(profile.ModeratorInfo.Languages) > MaxListItems {
			return coreiface.ErrTooManyItems{"moderatorinfo.languages", strconv.Itoa(MaxListItems)}
		}
		for _, l := range profile.ModeratorInfo.Languages {
			if len(l) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"moderatorinfo.languages", strconv.Itoa(WordMaxCharacters)}
			}
		}
		for _, l := range profile.ModeratorInfo.AcceptedCurrencies {
			if len(l) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"moderatorinfo.acceptedCurrencies", strconv.Itoa(WordMaxCharacters)}
			}
		}
		if len(profile.ModeratorInfo.AcceptedCurrencies) > MaxListItems {
			return coreiface.ErrTooManyItems{"moderatorinfo.acceptedCurrencies"}
		}
		if profile.ModeratorInfo.Fee.FixedFee != nil {
			if len(profile.ModeratorInfo.Fee.FixedFee.Currency.Name) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.name", strconv.Itoa(WordMaxCharacters)}
			}
			if len(string(profile.ModeratorInfo.Fee.FixedFee.Currency.CurrencyType)) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.currencytype", strconv.Itoa(WordMaxCharacters)}
			}
			if len(profile.ModeratorInfo.Fee.FixedFee.Currency.Code.String()) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.code", strconv.Itoa(WordMaxCharacters)}
			}
		}
	}
	if profile.AvatarHashes.Large != "" || profile.AvatarHashes.Medium != "" ||
		profile.AvatarHashes.Small != "" || profile.AvatarHashes.Tiny != "" || profile.AvatarHashes.Original != "" {
		_, err := cid.Decode(profile.AvatarHashes.Tiny)
		if err != nil {
			return errors.New("tiny image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.AvatarHashes.Small)
		if err != nil {
			return errors.New("small image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.AvatarHashes.Medium)
		if err != nil {
			return errors.New("medium image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.AvatarHashes.Large)
		if err != nil {
			return errors.New("large image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.AvatarHashes.Original)
		if err != nil {
			return errors.New("original image hashes must be properly formatted CID")
		}
	}
	if profile.HeaderHashes.Large != "" || profile.HeaderHashes.Medium != "" ||
		profile.HeaderHashes.Small != "" || profile.HeaderHashes.Tiny != "" || profile.HeaderHashes.Original != "" {
		_, err := cid.Decode(profile.HeaderHashes.Tiny)
		if err != nil {
			return errors.New("tiny image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.HeaderHashes.Small)
		if err != nil {
			return errors.New("small image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.HeaderHashes.Medium)
		if err != nil {
			return errors.New("medium image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.HeaderHashes.Large)
		if err != nil {
			return errors.New("large image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(profile.HeaderHashes.Original)
		if err != nil {
			return errors.New("original image hashes must be properly formatted CID")
		}
	}
	if len(profile.StoreAndForwardServers) > MaxListItems {
		return coreiface.ErrTooManyItems{"storeAndForwardServers"}
	}
	for _, pid := range profile.StoreAndForwardServers {
		_, err := peer.Decode(pid)
		if err != nil {
			return errors.New("invalid snf server peerID")
		}
	}
	if len(profile.EscrowPublicKey) != 66 {
		return fmt.Errorf("secp256k1 public key character length is greater than the max of %d", 66)
	}
	if profile.Stats != nil {
		if profile.Stats.AverageRating > 5 {
			return fmt.Errorf("average rating cannot be greater than %d", 5)
		}
	}
	return nil
}

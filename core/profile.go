package core

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
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
	pubkey, err := n.masterPrivKey.ECPubKey()
	if err != nil {
		return err
	}

	profile.PublicKey = hex.EncodeToString(pubkey.SerializeCompressed())
	profile.PeerID = n.ipfsNode.Identity.Pretty()
	profile.LastModified = time.Now()

	if err := validateProfile(profile); err != nil {
		return err
	}

	if err := n.updateProfileStats(profile); err != nil {
		return err
	}

	// TODO: add accepted currencies if moderator

	if err := n.repo.PublicData().SetProfile(profile); err != nil {
		return err
	}
	n.Publish(done)
	return nil
}

// GetMyProfile returns the profile for this node.
func (n *OpenBazaarNode) GetMyProfile() (*models.Profile, error) {
	return n.repo.PublicData().GetProfile()
}

// GetProfile returns the profile of the node with the given peer ID.
// If useCache is set it will return a profile from the local cache
// (if it has one) if profile is not found on the network.
func (n *OpenBazaarNode) GetProfile(peerID peer.ID, useCache bool) (*models.Profile, error) {
	pth, err := n.resolve(peerID, useCache)
	if err != nil {
		return nil, err
	}
	profileBytes, err := n.cat(path.Join(pth, repo.ProfileFile))
	if err != nil {
		return nil, err
	}
	profile := new(models.Profile)
	if err := json.Unmarshal(profileBytes, profile); err != nil {
		return nil, err
	}
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// updateAndSaveProfile loads the profile from disk, updates
// the profile stats, then saves it back to disk.
func (n *OpenBazaarNode) updateAndSaveProfile() error {
	profile, err := n.GetMyProfile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if profile == nil {
		return nil
	}
	if err := n.updateProfileStats(profile); err != nil {
		return err
	}
	return n.repo.PublicData().SetProfile(profile)
}

// updateProfileStats updates all stats on the passed in profile
func (n *OpenBazaarNode) updateProfileStats(profile *models.Profile) error {
	followers, err := n.repo.PublicData().GetFollowers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	following, err := n.repo.PublicData().GetFollowing()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	listings, err := n.repo.PublicData().GetListingIndex()
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

// validateProfile checks each field to make sure they're formatted properly and/or
// within the desired limits.
func validateProfile(profile *models.Profile) error {
	if len(profile.Name) == 0 {
		return ErrMissingField("name")
	}
	if len(profile.Name) > WordMaxCharacters {
		return ErrTooManyCharacters{"name", strconv.Itoa(WordMaxCharacters)}
	}
	if len(profile.Location) > WordMaxCharacters {
		return ErrTooManyCharacters{"location", strconv.Itoa(WordMaxCharacters)}
	}
	if len(profile.About) > AboutMaxCharacters {
		return ErrTooManyCharacters{"about", strconv.Itoa(AboutMaxCharacters)}
	}
	if len(profile.ShortDescription) > models.ShortDescriptionLength {
		return ErrTooManyCharacters{"shortdescription", strconv.Itoa(models.ShortDescriptionLength)}
	}
	if profile.ContactInfo != nil {
		if len(profile.ContactInfo.Website) > URLMaxCharacters {
			return ErrTooManyCharacters{"contactinfo.website", strconv.Itoa(URLMaxCharacters)}
		}
		if len(profile.ContactInfo.Email) > SentenceMaxCharacters {
			return ErrTooManyCharacters{"contactinfo.email", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(profile.ContactInfo.PhoneNumber) > WordMaxCharacters {
			return ErrTooManyCharacters{"contactinfo.phonenumber", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(profile.ContactInfo.Social) > MaxListItems {
			return ErrTooManyItems{"contactinfo.social", strconv.Itoa(MaxListItems)}
		}
		for _, s := range profile.ContactInfo.Social {
			if len(s.Username) > WordMaxCharacters {
				return ErrTooManyCharacters{"contactinfo.social.username", strconv.Itoa(WordMaxCharacters)}
			}
			if len(s.Type) > WordMaxCharacters {
				return ErrTooManyCharacters{"contactinfo.social.type", strconv.Itoa(WordMaxCharacters)}
			}
			if len(s.Proof) > URLMaxCharacters {
				return ErrTooManyCharacters{"contactinfo.social.proof", strconv.Itoa(URLMaxCharacters)}
			}
		}
	}
	if profile.ModeratorInfo != nil {
		if len(profile.ModeratorInfo.Description) > AboutMaxCharacters {
			return ErrTooManyCharacters{"moderatorinfo.description", strconv.Itoa(AboutMaxCharacters)}
		}
		if len(profile.ModeratorInfo.TermsAndConditions) > PolicyMaxCharacters {
			return ErrTooManyCharacters{"moderatorinfo.termsandconditions", strconv.Itoa(PolicyMaxCharacters)}
		}
		if len(profile.ModeratorInfo.Languages) > MaxListItems {
			return ErrTooManyItems{"moderatorinfo.languages", strconv.Itoa(MaxListItems)}
		}
		for _, l := range profile.ModeratorInfo.Languages {
			if len(l) > WordMaxCharacters {
				return ErrTooManyCharacters{"moderatorinfo.languages", strconv.Itoa(WordMaxCharacters)}
			}
		}
		if profile.ModeratorInfo.Fee.FixedFee != nil {
			if len(profile.ModeratorInfo.Fee.FixedFee.Currency.Name) > WordMaxCharacters {
				return ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.name", strconv.Itoa(WordMaxCharacters)}
			}
			if len(string(profile.ModeratorInfo.Fee.FixedFee.Currency.CurrencyType)) > WordMaxCharacters {
				return ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.currencytype", strconv.Itoa(WordMaxCharacters)}
			}
			if len(profile.ModeratorInfo.Fee.FixedFee.Currency.Code.String()) > WordMaxCharacters {
				return ErrTooManyCharacters{"moderatorinfo.fee.fixedfee.currency.code", strconv.Itoa(WordMaxCharacters)}
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
	if len(profile.PublicKey) != 66 {
		return fmt.Errorf("secp256k1 public key character length is greater than the max of %d", 66)
	}
	if profile.Stats != nil {
		if profile.Stats.AverageRating > 5 {
			return fmt.Errorf("average rating cannot be greater than %d", 5)
		}
	}
	return nil
}

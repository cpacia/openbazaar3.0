package core

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

const (
	profileFile = "profile.json"
)

// SetProfile sets the public profile for the node and publishes to IPNS.
//
// The publishing is done in a separate goroutine so this function will
// return as soon as the profile is saved to disk. The optional done
// chan will be closed when publishing is complete.
func (n *OpenBazaarNode) SetProfile(profile *models.Profile, done chan<- struct{}) error {
	if err := validateProfile(profile); err != nil {
		return err
	}

	pubkey, err := n.masterPrivKey.ECPubKey()
	if err != nil {
		return err
	}

	profile.PublicKey = hex.EncodeToString(pubkey.SerializeCompressed())
	profile.PeerID = n.ipfsNode.Identity.Pretty()
	profile.LastModified = time.Now()

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
// If checkCache is set it will return a profile from the local cache
// (if it has one) if profile is not found on the network.
func (n *OpenBazaarNode) GetProfile(peerID peer.ID, fromCache bool) (*models.Profile, error) {
	pth, err := n.resolve(peerID, fromCache)
	if err != nil {
		return nil, err
	}
	profileBytes, err := n.cat(path.Join(pth, profileFile))
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

// validateProfile checks each field to make sure they're formatted properly and/or
// within the desired limits.
func validateProfile(profile *models.Profile) error {
	if len(profile.Name) == 0 {
		return errors.New("profile name not set")
	}
	if len(profile.Name) > WordMaxCharacters {
		return fmt.Errorf("name character length is greater than the max of %d", WordMaxCharacters)
	}
	if len(profile.Location) > WordMaxCharacters {
		return fmt.Errorf("location character length is greater than the max of %d", WordMaxCharacters)
	}
	if len(profile.About) > AboutMaxCharacters {
		return fmt.Errorf("about character length is greater than the max of %d", AboutMaxCharacters)
	}
	if len(profile.ShortDescription) > ShortDescriptionLength {
		return fmt.Errorf("short description character length is greater than the max of %d", ShortDescriptionLength)
	}
	if profile.ContactInfo != nil {
		if len(profile.ContactInfo.Website) > URLMaxCharacters {
			return fmt.Errorf("website character length is greater than the max of %d", URLMaxCharacters)
		}
		if len(profile.ContactInfo.Email) > SentenceMaxCharacters {
			return fmt.Errorf("email character length is greater than the max of %d", SentenceMaxCharacters)
		}
		if len(profile.ContactInfo.PhoneNumber) > WordMaxCharacters {
			return fmt.Errorf("phone number character length is greater than the max of %d", WordMaxCharacters)
		}
		if len(profile.ContactInfo.Social) > MaxListItems {
			return fmt.Errorf("number of social accounts is greater than the max of %d", MaxListItems)
		}
		for _, s := range profile.ContactInfo.Social {
			if len(s.Username) > WordMaxCharacters {
				return fmt.Errorf("social username character length is greater than the max of %d", WordMaxCharacters)
			}
			if len(s.Type) > WordMaxCharacters {
				return fmt.Errorf("social account type character length is greater than the max of %d", WordMaxCharacters)
			}
			if len(s.Proof) > URLMaxCharacters {
				return fmt.Errorf("social proof character length is greater than the max of %d", WordMaxCharacters)
			}
		}
	}
	if profile.ModeratorInfo != nil {
		if len(profile.ModeratorInfo.Description) > AboutMaxCharacters {
			return fmt.Errorf("moderator description character length is greater than the max of %d", AboutMaxCharacters)
		}
		if len(profile.ModeratorInfo.TermsAndConditions) > PolicyMaxCharacters {
			return fmt.Errorf("moderator terms and conditions character length is greater than the max of %d", PolicyMaxCharacters)
		}
		if len(profile.ModeratorInfo.Languages) > MaxListItems {
			return fmt.Errorf("moderator number of languages greater than the max of %d", MaxListItems)
		}
		for _, l := range profile.ModeratorInfo.Languages {
			if len(l) > WordMaxCharacters {
				return fmt.Errorf("moderator language character length is greater than the max of %d", WordMaxCharacters)
			}
		}
		if profile.ModeratorInfo.Fee.FixedFee != nil {
			if len(profile.ModeratorInfo.Fee.FixedFee.CurrencyCode) > WordMaxCharacters {
				return fmt.Errorf("moderator fee currency code character length is greater than the max of %d", WordMaxCharacters)
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
	if len(profile.PublicKey) > 66 {
		return fmt.Errorf("bitcoin public key character length is greater than the max of %d", 66)
	}
	if profile.Stats != nil {
		if profile.Stats.AverageRating > 5 {
			return fmt.Errorf("average rating cannot be greater than %d", 5)
		}
	}
	return nil
}

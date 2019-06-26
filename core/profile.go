package core

import (
	"encoding/hex"
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

const (
	profileFile = "profile.json"
)

// SetProfile sets the public profile for the node and publishes to IPNS.
func (n *OpenBazaarNode) SetProfile(profile *models.Profile) error {
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
	return n.Publish()
}

// GetMyProfile returns the profile for this node.
func (n *OpenBazaarNode) GetMyProfile() (*models.Profile, error) {
	return n.repo.PublicData().GetProfile()
}

// GetProfile returns the profile of the node with the given peer ID.
// If checkCache is set it will return a profile from the local cache
// (if it has one) if profile is not found on the network.
func (n *OpenBazaarNode) GetProfile(peerID peer.ID, checkCache bool) (*models.Profile, error) {
	pth, err := n.resolve(peerID, checkCache)
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
	return profile, nil
}

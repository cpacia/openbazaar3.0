package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/jsonpb"
	"github.com/ipfs/go-cid"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Returns the rating index file for this node.
func (n *OpenBazaarNode) GetMyRatings() (models.RatingIndex, error) {
	var (
		index models.RatingIndex
		err   error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		index, err = tx.GetRatingIndex()
		if err != nil {
			return fmt.Errorf("%w: rating index not found", coreiface.ErrNotFound)
		}
		return nil
	})
	return index, err
}

// GetRatings returns the rating index for node with the given peer ID.
// If useCache is set it will return the index from the local cache
// (if it has one) if rating index file is not found on the network.
func (n *OpenBazaarNode) GetRatings(ctx context.Context, peerID peer.ID, useCache bool) (models.RatingIndex, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	indexBytes, err := n.cat(ctx, ipath.Join(pth, ffsqlite.RatingIndexFile))
	if err != nil {
		return nil, err
	}
	var index models.RatingIndex
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		return nil, err
	}
	return index, nil
}

// GetRating fetches the rating from the network given its cid. It will attempt to validating
// the rating using the signatures embedded in the rating. If they are invalid an error will
// be returned.
func (n *OpenBazaarNode) GetRating(ctx context.Context, cid cid.Cid) (*pb.Rating, error) {
	ratingBytes, err := n.cat(ctx, ipath.IpfsPath(cid))
	if err != nil {
		return nil, err
	}
	var rating pb.Rating
	if err := jsonpb.UnmarshalString(string(ratingBytes), &rating); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	if err := utils.ValidateRating(&rating); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	return &rating, nil
}

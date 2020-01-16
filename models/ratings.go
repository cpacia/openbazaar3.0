package models

import (
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/ipfs/go-cid"
)

// RatingIndex is a list of RatingInfo objects. It is saved
// in the public data directory.
type RatingIndex []RatingInfo

// AddRating adds a new rating to the index. If the cid already exists, it
// will not be added to the index.
func (ri *RatingIndex) AddRating(rating *pb.Rating, cid cid.Cid) error {
	var (
		ratingInfo RatingInfo
		index      = -1
	)
	for i, info := range *ri {
		if info.Slug == rating.VendorSig.Slug {
			ratingInfo = info
			index = i
			break
		}
	}
	for _, id := range ratingInfo.Ratings {
		if id == cid.String() {
			return nil // Already exists
		}
	}

	total := float64(ratingInfo.Count) * ratingInfo.Average
	total += float64(rating.Overall)
	newAverage := total / float64(ratingInfo.Count+1)

	ratingInfo.Average = newAverage
	ratingInfo.Count++
	ratingInfo.Slug = rating.VendorSig.Slug
	ratingInfo.Ratings = append(ratingInfo.Ratings, cid.String())
	if index == -1 {
		*ri = append(*ri, ratingInfo)
	} else {
		(*ri)[index] = ratingInfo
	}
	return nil
}

// RatingInfo stores info about the ratings for each listing.
type RatingInfo struct {
	Slug    string   `json:"slug"`
	Count   uint64   `json:"count"`
	Average float64  `json:"average"`
	Ratings []string `json:"ratings"`
}

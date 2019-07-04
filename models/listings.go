package models

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/ipfs/go-cid"
	"math/big"
)

const (
	// ShortDescriptionLength is the maximum length of the short description.
	ShortDescriptionLength = 160
)

// ListingIndex is a list of metadata objects. It is saved
// in the public data directory.
type ListingIndex []ListingMetadata

// UpdateListing will replace the existing metadata object in the index
// with the provided metadata. If the listing does not exist in the index
// then the metadata object will be appended.
func (li *ListingIndex) UpdateListing(listingMetadata ListingMetadata) {
	exists := false
	for i, lm := range *li {
		if lm.Slug == listingMetadata.Slug {
			(*li)[i] = listingMetadata
			exists = true
			break
		}
	}
	if !exists {
		*li = append(*li, listingMetadata)
	}
}

func (li *ListingIndex) DeleteListing(slug string) {
	for i, lm := range *li {
		if lm.Slug == slug {
			*li = append((*li)[:i], (*li)[i+1:]...)
			break
		}
	}
}

func (li *ListingIndex) GetListingSlug(cid cid.Cid) (string, error) {
	for _, lm := range *li {
		if lm.Hash == cid.String() {
			return lm.Slug, nil
		}
	}
	return "", errors.New("listing not found")
}

// Count returns the number of listings.
func (li *ListingIndex) Count() int {
	return len(*li)
}

// ListingMetadata is the metadata for an individual listing.
// The node's listing index is an array of these objects.
type ListingMetadata struct {
	Hash               string           `json:"hash"`
	Slug               string           `json:"slug"`
	Title              string           `json:"title"`
	Categories         []string         `json:"categories"`
	NSFW               bool             `json:"nsfw"`
	ContractType       string           `json:"contractType"`
	Description        string           `json:"description"`
	Thumbnail          ListingThumbnail `json:"thumbnail"`
	Price              CurrencyValue    `json:"price"`
	ShipsTo            []string         `json:"shipsTo"`
	FreeShipping       []string         `json:"freeShipping"`
	Language           string           `json:"language"`
	AverageRating      float32          `json:"averageRating"`
	RatingCount        uint32           `json:"ratingCount"`
	ModeratorIDs       []string         `json:"moderators"`
	AcceptedCurrencies []string         `json:"acceptedCurrencies"`
	CoinType           string           `json:"coinType"`
}

// NewListingMetadataFromListing returns a new ListingMetadata object given a
// pb.Listing and its cid.
func NewListingMetadataFromListing(listing *pb.Listing, cid cid.Cid) (*ListingMetadata, error) {
	descriptionLength := len(listing.Item.Description)
	if descriptionLength > ShortDescriptionLength {
		descriptionLength = ShortDescriptionLength
	}

	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}

	var shipsTo []string
	var freeShipping []string
	for _, shippingOption := range listing.ShippingOptions {
		for _, region := range shippingOption.Regions {
			if !contains(shipsTo, region.String()) {
				shipsTo = append(shipsTo, region.String())
			}
			for _, service := range shippingOption.Services {
				amt, _ := new(big.Int).SetString(service.Price, 10)
				if amt.Cmp(big.NewInt(0)) == 0 && !contains(freeShipping, region.String()) {
					freeShipping = append(freeShipping, region.String())
				}
			}
		}
	}

	cv, err := NewCurrencyValue(listing.Item.Price, CurrencyDefinitions[listing.Metadata.PricingCurrency.Code])
	if err != nil {
		return nil, err
	}

	ld := &ListingMetadata{
		Hash:         cid.String(),
		Slug:         listing.Slug,
		Title:        listing.Item.Title,
		Categories:   listing.Item.Categories,
		NSFW:         listing.Item.Nsfw,
		CoinType:     listing.Metadata.PricingCurrency.Code,
		ContractType: listing.Metadata.ContractType.String(),
		Description:  listing.Item.Description[:descriptionLength],
		Thumbnail: ListingThumbnail{
			Tiny:   listing.Item.Images[0].Tiny,
			Small:  listing.Item.Images[0].Small,
			Medium: listing.Item.Images[0].Medium,
		},
		Price:              *cv,
		ShipsTo:            shipsTo,
		FreeShipping:       freeShipping,
		Language:           listing.Metadata.Language,
		ModeratorIDs:       listing.Moderators,
		AcceptedCurrencies: listing.Metadata.AcceptedCurrencies,
	}
	return ld, nil
}

// ListingThumbnail holds the thumbnail hashes for a listing.
type ListingThumbnail struct {
	Tiny   string `json:"tiny"`
	Small  string `json:"small"`
	Medium string `json:"medium"`
}

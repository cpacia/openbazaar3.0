package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenBazaar/jsonpb"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/database/ffsqlite"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/proto"
	"github.com/gosimple/slug"
	"github.com/ipfs/go-cid"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/microcosm-cc/bluemonday"
	"github.com/multiformats/go-multihash"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// SaveListing saves the provided listing. It will validate it
// to make sure it conforms to the requirements, update the listing
// index and update the listing count in the profile.
func (n *OpenBazaarNode) SaveListing(listing *pb.Listing, done chan<- struct{}) error {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		cid, err := n.saveListingToDB(tx, listing)
		if err != nil {
			return err
		}

		lmd, err := models.NewListingMetadataFromListing(listing, cid)
		if err != nil {
			return err
		}

		index, err := tx.GetListingIndex()
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		index.UpdateListing(*lmd)

		if err := tx.SetListingIndex(index); err != nil {
			return err
		}

		// Update profile counts
		if err := n.updateAndSaveProfile(tx); err != nil {
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

// UpdateAllListings will iterate over each listing and pass it in to updateFunc.
// The function should update the listing point in place and return a boolean
// expressing whether or not the listing was updated.
func (n *OpenBazaarNode) UpdateAllListings(updateFunc func(l *pb.Listing) (bool, error), done chan<- struct{}) error {
	var (
		listingsUpdated = false
		err             error
	)
	err = n.repo.DB().Update(func(tx database.Tx) error {
		listingsUpdated, err = n.updateAllListings(tx, updateFunc)
		return err
	})
	if err != nil {
		maybeCloseDone(done)
		return err
	}
	if !listingsUpdated {
		maybeCloseDone(done)
		return nil
	}

	n.Publish(done)
	return nil
}

// DeleteListing deletes the listing from disk, updates the listing index and
// profile counts, and publishes.
func (n *OpenBazaarNode) DeleteListing(slug string, done chan<- struct{}) error {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		if err := tx.Delete("slug", slug, nil, &models.Coupon{}); err != nil {
			return err
		}

		index, err := tx.GetListingIndex()
		if err != nil {
			return fmt.Errorf("%w: listing index not found", coreiface.ErrNotFound)
		}
		index.DeleteListing(slug)
		if err := tx.SetListingIndex(index); err != nil {
			return err
		}

		if err := tx.DeleteListing(slug); err != nil {
			return fmt.Errorf("%w: listing not found", coreiface.ErrNotFound)
		}

		if err := n.updateAndSaveProfile(tx); err != nil {
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

// Returns the listing index file for this node.
func (n *OpenBazaarNode) GetMyListings() (models.ListingIndex, error) {
	var (
		index models.ListingIndex
		err   error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		index, err = tx.GetListingIndex()
		if err != nil {
			return fmt.Errorf("%w: listing index not found", coreiface.ErrNotFound)
		}
		return nil
	})
	return index, err
}

// GetListings returns the listing index for node with the given peer ID.
// If useCache is set it will return the index from the local cache
// (if it has one) if listing index file is not found on the network.
func (n *OpenBazaarNode) GetListings(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	indexBytes, err := n.cat(ctx, ipath.Join(pth, ffsqlite.ListingIndexFile))
	if err != nil {
		return nil, err
	}
	var index models.ListingIndex
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		return nil, err
	}
	return index, nil
}

// GetMyListingBySlug returns our own listing given the slug.
func (n *OpenBazaarNode) GetMyListingBySlug(slug string) (*pb.SignedListing, error) {
	var (
		listing *pb.SignedListing
		coupons []models.Coupon
		err     error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		index, err := tx.GetListingIndex()
		if err != nil {
			return fmt.Errorf("%w: listing index not found", coreiface.ErrNotFound)
		}

		id, err := index.GetListingCID(slug)
		if err != nil {
			return fmt.Errorf("%w: listing not found", coreiface.ErrNotFound)
		}

		listing, err = tx.GetListing(slug)
		if err != nil {
			return fmt.Errorf("%w: listing not found", coreiface.ErrNotFound)
		}
		if err := tx.Read().Where("slug = ?", slug).Find(&coupons).Error; !gorm.IsRecordNotFoundError(err) {
			return err
		}

		listing.Cid = id.String()
		return nil
	})
	if err != nil {
		return nil, err
	}

	if listing.Listing.Coupons != nil {
		swapCouponHashesWithDiscountCodes(listing, coupons)
	}
	return listing, nil
}

// GetMyListingByCID returns our own listing given the cid.
func (n *OpenBazaarNode) GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error) {
	var (
		listing *pb.SignedListing
		coupons []models.Coupon
		err     error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		index, err := tx.GetListingIndex()
		if err != nil {
			return fmt.Errorf("%w: listing index not found", coreiface.ErrNotFound)
		}
		slug, err := index.GetListingSlug(cid)
		if err != nil {
			return fmt.Errorf("%w: listing not found in index", coreiface.ErrNotFound)
		}
		listing, err = tx.GetListing(slug)
		if err != nil {
			return fmt.Errorf("%w: listing not found", coreiface.ErrNotFound)
		}
		if err := tx.Read().Where("slug = ?", slug).Find(&coupons).Error; !gorm.IsRecordNotFoundError(err) {
			return err
		}
		listing.Cid = cid.String()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if listing.Listing.Coupons != nil {
		swapCouponHashesWithDiscountCodes(listing, coupons)
	}
	return listing, nil
}

// GetListingBySlug returns a listing for node with the given peer ID.
// If useCache is set it will return the listing from the local cache
// (if it has one) if listing file is not found on the network.
func (n *OpenBazaarNode) GetListingBySlug(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}
	listingBytes, err := n.cat(ctx, ipath.Join(pth, "listings", slug+".json"))
	if err != nil {
		return nil, err
	}
	cid, err := n.cid(listingBytes)
	if err != nil {
		return nil, err
	}
	return n.deserializeAndValidate(listingBytes, cid)
}

// GetListingByCID fetches the listing from the network given its cid.
func (n *OpenBazaarNode) GetListingByCID(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error) {
	listingBytes, err := n.cat(ctx, ipath.IpfsPath(cid))
	if err != nil {
		return nil, err
	}
	return n.deserializeAndValidate(listingBytes, cid)
}

// generateListingSlug generates a slug from the title of the listing. It
// makes sure the slug is not used by any other listing. If it is it will
// add an integer to the end of the slug and increment if necessary.
func (n *OpenBazaarNode) generateListingSlug(title string) (string, error) {
	title = strings.Replace(title, "/", "", -1)
	counter := 1

	l := SentenceMaxCharacters - SlugBuffer

	var rx = regexp.MustCompile(EmojiPattern)
	title = rx.ReplaceAllStringFunc(title, func(s string) string {
		r, _ := utf8.DecodeRuneInString(s)
		html := fmt.Sprintf(`&#x%X;`, r)
		return html
	})

	slugBase := slug.Make(title)
	if len(slugBase) < SentenceMaxCharacters-SlugBuffer {
		l = len(slugBase)
	}
	slugBase = slugBase[:l]

	slugToTry := slugBase
	for {
		_, err := n.GetMyListingBySlug(slugToTry)
		if errors.Is(err, coreiface.ErrNotFound) {
			return slugToTry, nil
		} else if err != nil {
			return "", err
		}
		slugToTry = slugBase + strconv.Itoa(counter)
		counter++
	}
}

func (n *OpenBazaarNode) updateAllListings(tx database.Tx, updateFunc func(l *pb.Listing) (bool, error)) (listingsUpdated bool, _ error) {
	index, err := tx.GetListingIndex()
	if err != nil {
		return false, err
	}

	var updatedMetadata []models.ListingMetadata
	for _, lmd := range index {
		signedListing, err := tx.GetListing(lmd.Slug)
		if err != nil {
			return false, err
		}
		listing := signedListing.Listing

		updated, err := updateFunc(listing)
		if err != nil {
			return false, err
		}

		if updated {
			cid, err := n.saveListingToDB(tx, listing)
			if err != nil {
				return false, err
			}

			newLmd, err := models.NewListingMetadataFromListing(listing, cid)
			if err != nil {
				return false, err
			}

			updatedMetadata = append(updatedMetadata, *newLmd)
			listingsUpdated = true
		}
	}
	if !listingsUpdated {
		return false, nil
	}

	for _, lmd := range updatedMetadata {
		index.UpdateListing(lmd)
	}

	// Save updated index.
	if err := tx.SetListingIndex(index); err != nil {
		return true, err
	}

	// Update profile counts
	return true, n.updateAndSaveProfile(tx)
}

// saveListingToDB updates any needed fields in the listing and saves or updates the
// listing on disk and the coupon database table.
func (n *OpenBazaarNode) saveListingToDB(dbtx database.Tx, listing *pb.Listing) (cid.Cid, error) {
	// Set the escrow timeout.
	if n.UsingTestnet() {
		// Testnet should be set to one hour unless otherwise
		// specified. This allows for easier testing.
		if listing.Metadata.EscrowTimeoutHours == 0 {
			listing.Metadata.EscrowTimeoutHours = 1
		}
	} else {
		listing.Metadata.EscrowTimeoutHours = EscrowTimeout
	}

	// If slug is not set create a new one.
	if listing.Slug == "" {
		var err error
		listing.Slug, err = n.generateListingSlug(listing.Item.Title)
		if err != nil {
			return cid.Cid{}, err
		}
	}

	currencyMap := make(map[string]bool)
	for _, acceptedCurrency := range listing.Metadata.AcceptedCurrencies {
		_, err := n.multiwallet.WalletForCurrencyCode(acceptedCurrency)
		if err != nil {
			return cid.Cid{}, fmt.Errorf("%w: currency %s is not found in multiwallet", coreiface.ErrBadRequest, acceptedCurrency)
		}
		if currencyMap[normalizeCurrencyCode(acceptedCurrency)] {
			return cid.Cid{}, fmt.Errorf("%w: duplicate accepted currency in listing", coreiface.ErrBadRequest)
		}
		currencyMap[normalizeCurrencyCode(acceptedCurrency)] = true
	}

	// Sanitize a few critical fields
	if listing.Item == nil {
		return cid.Cid{}, fmt.Errorf("%w: no item in listing", coreiface.ErrBadRequest)
	}
	sanitizer := bluemonday.UGCPolicy()
	for _, opt := range listing.Item.Options {
		opt.Name = sanitizer.Sanitize(opt.Name)
		for _, v := range opt.Variants {
			v.Name = sanitizer.Sanitize(v.Name)
		}
	}
	for _, so := range listing.ShippingOptions {
		so.Name = sanitizer.Sanitize(so.Name)
		for _, serv := range so.Services {
			serv.Name = sanitizer.Sanitize(serv.Name)
		}
	}

	// Set listing version
	if listing.Metadata.Version <= 0 {
		listing.Metadata.Version = ListingVersion
	}

	// Add the vendor ID to the listing
	profile, err := dbtx.GetProfile()
	if err != nil && !os.IsNotExist(err) {
		return cid.Cid{}, err
	}
	pubkey, err := crypto.MarshalPublicKey(n.ipfsNode.PrivateKey.GetPublic())
	if err != nil {
		return cid.Cid{}, err
	}

	idHash := sha256.Sum256([]byte(n.Identity().Pretty()))
	sig, err := n.escrowMasterKey.Sign(idHash[:])
	if err != nil {
		return cid.Cid{}, err
	}
	listing.VendorID = &pb.ID{
		PeerID: n.Identity().Pretty(),
		Pubkeys: &pb.ID_Pubkeys{
			Identity: pubkey,
			Escrow:   n.escrowMasterKey.PubKey().SerializeCompressed(),
		},
		Sig: sig.Serialize(),
	}
	if profile != nil {
		listing.VendorID.Handle = profile.Handle
	}

	var couponsToStore []models.Coupon
	for i, coupon := range listing.Coupons {
		hash := coupon.GetHash()
		code := coupon.GetDiscountCode()

		_, err := multihash.FromB58String(hash)
		if err != nil {
			couponMH, err := utils.MultihashSha256([]byte(code))
			if err != nil {
				return cid.Cid{}, err
			}

			listing.Coupons[i].Code = &pb.Listing_Coupon_Hash{Hash: couponMH.B58String()}
			hash = couponMH.B58String()
		}
		coupon := models.Coupon{Slug: listing.Slug, Code: code, Hash: hash}
		couponsToStore = append(couponsToStore, coupon)
	}
	if err := dbtx.Delete("slug", listing.Slug, nil, &models.Coupon{}); err != nil {
		return cid.Cid{}, err
	}
	if len(couponsToStore) > 0 {
		for _, coupon := range couponsToStore {
			if err := dbtx.Save(&coupon); err != nil {
				return cid.Cid{}, err
			}
		}
	}

	// Sign listing
	sl, err := n.signListing(listing)
	if err != nil {
		return cid.Cid{}, err
	}

	// Check the listing data is correct for continuing
	if err := n.validateListing(sl); err != nil {
		if errors.Is(err, coreiface.ErrInternalServer) {
			return cid.Cid{}, err
		} else {
			return cid.Cid{}, fmt.Errorf("%w: %s", coreiface.ErrBadRequest, err)
		}
	}

	// Save listing
	if err := dbtx.SetListing(sl); err != nil {
		return cid.Cid{}, err
	}

	m := jsonpb.Marshaler{
		Indent:       "    ",
		EmitDefaults: false,
	}
	ser, err := m.MarshalToString(sl)
	if err != nil {
		return cid.Cid{}, err
	}

	// Update listing index
	return n.cid([]byte(ser))
}

// signListing signs a protobuf serialization of the listing with the inventory
// zeroed out (see serializeSignatureFormat below).
func (n *OpenBazaarNode) signListing(listing *pb.Listing) (*pb.SignedListing, error) {
	ser, err := proto.Marshal(listing)
	if err != nil {
		return nil, err
	}
	sig, err := n.ipfsNode.PrivateKey.Sign(ser)
	if err != nil {
		return nil, err
	}
	return &pb.SignedListing{Listing: listing, Signature: sig}, nil
}

// removeDisabledCoinsFromListings loops over all listings and removes any accepted
// currencies that are listed for a wallet that is not enabled by this node.
func (n *OpenBazaarNode) removeDisabledCoinsFromListings() error {
	enabledCoins := make(map[string]bool)
	for ct := range n.multiwallet {
		enabledCoins[ct.CurrencyCode()] = true
	}

	return n.UpdateAllListings(func(listing *pb.Listing) (bool, error) {
		var (
			newAcceptedCurrencies = make([]string, 0, len(listing.Metadata.AcceptedCurrencies))
			updated               = false
		)

		for _, acceptedCurrency := range listing.Metadata.AcceptedCurrencies {
			if enabledCoins[acceptedCurrency] {
				newAcceptedCurrencies = append(newAcceptedCurrencies, acceptedCurrency)
			} else {
				updated = true
			}
		}
		if updated {
			listing.Metadata.AcceptedCurrencies = newAcceptedCurrencies
		}
		return updated, nil
	}, nil)
}

// validateListing performs a ton of checks to make sure the listing is formatted correctly.
// We should not allow invalid listings to be saved or purchased as it can lead to ambiguity
// when moderating a dispute or possible attacks. This function needs to be maintained in
// conjunction with the listing.proto.
func (n *OpenBazaarNode) validateListing(sl *pb.SignedListing) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()

	// Slug
	if sl.Listing.Slug == "" {
		return coreiface.ErrMissingField("slug")
	}
	if len(sl.Listing.Slug) > SentenceMaxCharacters {
		return coreiface.ErrTooManyCharacters{"slug", strconv.Itoa(SentenceMaxCharacters)}
	}
	if strings.Contains(sl.Listing.Slug, " ") {
		return errors.New("slugs cannot contain spaces")
	}
	if strings.Contains(sl.Listing.Slug, "/") {
		return errors.New("slugs cannot contain file separators")
	}

	// Metadata
	if sl.Listing.Metadata == nil {
		return coreiface.ErrMissingField("metadata")
	}
	if sl.Listing.Metadata.ContractType > pb.Listing_Metadata_CRYPTOCURRENCY {
		return errors.New("invalid contract type")
	}
	if sl.Listing.Metadata.Format > pb.Listing_Metadata_MARKET_PRICE {
		return errors.New("invalid listing format")
	}
	if sl.Listing.Metadata.Expiry == nil {
		return coreiface.ErrMissingField("metadata.expiry")
	}
	if time.Unix(sl.Listing.Metadata.Expiry.Seconds, 0).Before(time.Now()) {
		return errors.New("listing expiration must be in the future")
	}
	if len(sl.Listing.Metadata.Language) > WordMaxCharacters {
		return coreiface.ErrTooManyCharacters{"metadata.language", strconv.Itoa(WordMaxCharacters)}
	}
	if !n.testnet && sl.Listing.Metadata.EscrowTimeoutHours > EscrowTimeout {
		return fmt.Errorf("escrow timeout must be less than or equal to %d hours", EscrowTimeout)
	}
	if len(sl.Listing.Metadata.AcceptedCurrencies) == 0 {
		return coreiface.ErrMissingField("metadata.acceptedcurrencies")
	}
	if len(sl.Listing.Metadata.AcceptedCurrencies) > MaxListItems {
		return coreiface.ErrTooManyItems{"metadata.acceptedcurrencies", strconv.Itoa(MaxListItems)}
	}
	for _, c := range sl.Listing.Metadata.AcceptedCurrencies {
		if len(c) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"metadata.acceptedcurrencies", strconv.Itoa(WordMaxCharacters)}
		}
	}
	if sl.Listing.Metadata.PricingCurrency == nil {
		return coreiface.ErrMissingField("metadata.pricingcurrency")
	}
	if sl.Listing.Metadata.PricingCurrency.Code == "" {
		return coreiface.ErrMissingField("metadata.pricingcurrency.code")
	}
	if len(sl.Listing.Metadata.PricingCurrency.Code) > WordMaxCharacters {
		return coreiface.ErrTooManyCharacters{"metadata.pricingcurrency.code", strconv.Itoa(WordMaxCharacters)}
	}
	def, err := models.CurrencyDefinitions.Lookup(sl.Listing.Metadata.PricingCurrency.Code)
	if err != nil {
		return errors.New("unknown pricing currency")
	}
	if sl.Listing.Metadata.PricingCurrency.Divisibility != uint32(def.Divisibility) {
		return errors.New("divisibility differs from expected value")
	}

	// Item
	if sl.Listing.Item.Title == "" {
		return coreiface.ErrMissingField("item.title")
	}
	price, _ := new(big.Int).SetString(sl.Listing.Item.Price, 10)
	if (sl.Listing.Metadata.ContractType != pb.Listing_Metadata_CRYPTOCURRENCY &&
		sl.Listing.Metadata.ContractType != pb.Listing_Metadata_CLASSIFIED) &&
		price.Cmp(big.NewInt(0)) == 0 {
		return errors.New("zero price listings are not allowed")
	}
	if sl.Listing.Metadata.ContractType == pb.Listing_Metadata_CLASSIFIED && len(sl.Listing.ShippingOptions) > 0 {
		return errors.New("classified listings can not have shipping")
	}
	if len(sl.Listing.Item.Title) > TitleMaxCharacters {
		return coreiface.ErrTooManyCharacters{"item.title", strconv.Itoa(TitleMaxCharacters)}
	}
	if len(sl.Listing.Item.Description) > DescriptionMaxCharacters {
		return coreiface.ErrTooManyCharacters{"item.description", strconv.Itoa(DescriptionMaxCharacters)}
	}
	if len(sl.Listing.Item.ProcessingTime) > SentenceMaxCharacters {
		return coreiface.ErrTooManyCharacters{"item.processingtime", strconv.Itoa(SentenceMaxCharacters)}
	}
	if len(sl.Listing.Item.Tags) > MaxTags {
		return fmt.Errorf("number of tags exceeds the max of %d", MaxTags)
	}
	for _, tag := range sl.Listing.Item.Tags {
		if tag == "" {
			return errors.New("tags must not be empty")
		}
		if len(tag) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.tags", strconv.Itoa(WordMaxCharacters)}
		}
	}
	if len(sl.Listing.Item.Images) == 0 {
		return coreiface.ErrMissingField("item.images")
	}
	if len(sl.Listing.Item.Images) > MaxListItems {
		return coreiface.ErrTooManyItems{"item.images", strconv.Itoa(MaxListItems)}
	}
	for _, img := range sl.Listing.Item.Images {
		_, err := cid.Decode(img.Tiny)
		if err != nil {
			return errors.New("tiny image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Small)
		if err != nil {
			return errors.New("small image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Medium)
		if err != nil {
			return errors.New("medium image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Large)
		if err != nil {
			return errors.New("large image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Original)
		if err != nil {
			return errors.New("original image hashes must be properly formatted CID")
		}
		if img.Filename == "" {
			return errors.New("image file names must not be nil")
		}
		if len(img.Filename) > FilenameMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.images.filename", strconv.Itoa(FilenameMaxCharacters)}
		}
	}
	if len(sl.Listing.Item.Categories) > MaxCategories {
		return fmt.Errorf("number of categories must be less than max of %d", MaxCategories)
	}
	for _, category := range sl.Listing.Item.Categories {
		if category == "" {
			return coreiface.ErrMissingField("item.category")
		}
		if len(category) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.categories", strconv.Itoa(WordMaxCharacters)}
		}
	}

	maxCombos := 1
	optionMap := make(map[string]map[string]struct{})
	for _, option := range sl.Listing.Item.Options {
		if _, ok := optionMap[option.Name]; ok {
			return errors.New("option names must be unique")
		}
		if option.Name == "" {
			return coreiface.ErrMissingField("item.options.name")
		}
		if len(option.Variants) < 2 {
			return errors.New("options must have more than one variants")
		}
		if len(option.Name) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.options.name", strconv.Itoa(WordMaxCharacters)}
		}
		if len(option.Description) > SentenceMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.options.description", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(option.Variants) > MaxListItems {
			return coreiface.ErrTooManyItems{"item.options.variants", strconv.Itoa(MaxListItems)}
		}
		varMap := make(map[string]struct{})
		for _, variant := range option.Variants {
			if _, ok := varMap[variant.Name]; ok {
				return errors.New("variant names must be unique")
			}
			if len(variant.Name) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"item.options.variants.name", strconv.Itoa(WordMaxCharacters)}
			}
			if variant.Image != nil && (variant.Image.Filename != "" ||
				variant.Image.Large != "" || variant.Image.Medium != "" || variant.Image.Small != "" ||
				variant.Image.Tiny != "" || variant.Image.Original != "") {
				_, err := cid.Decode(variant.Image.Tiny)
				if err != nil {
					return errors.New("tiny image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Small)
				if err != nil {
					return errors.New("small image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Medium)
				if err != nil {
					return errors.New("medium image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Large)
				if err != nil {
					return errors.New("large image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Original)
				if err != nil {
					return errors.New("original image hashes must be properly formatted CID")
				}
				if variant.Image.Filename == "" {
					return coreiface.ErrMissingField("items.options.variants.image.file")
				}
				if len(variant.Image.Filename) > FilenameMaxCharacters {
					return coreiface.ErrTooManyCharacters{"item.options.variants.image.filename", strconv.Itoa(FilenameMaxCharacters)}
				}
			}
			varMap[variant.Name] = struct{}{}
		}
		maxCombos *= len(option.Variants)
		optionMap[option.Name] = varMap
	}

	if len(sl.Listing.Item.Skus) > maxCombos {
		return errors.New("more skus than variant combinations")
	}
	comboMap := make(map[string]bool)
	for _, sku := range sl.Listing.Item.Skus {
		if maxCombos > 1 && len(sku.Selections) == 0 {
			return errors.New("skus must specify a variant combo when options are used")
		}
		if len(sku.ProductID) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"item.sku.productID", strconv.Itoa(WordMaxCharacters)}
		}
		formatted, err := json.Marshal(sku.Selections)
		if err != nil {
			return err
		}
		_, ok := comboMap[string(formatted)]
		if !ok {
			comboMap[string(formatted)] = true
		} else {
			return errors.New("duplicate sku")
		}
		if len(sku.Selections) != len(sl.Listing.Item.Options) {
			return errors.New("incorrect number of variants in sku combination")
		}
		for _, selection := range sku.Selections {
			variantMap, ok := optionMap[selection.Option]
			if !ok {
				return errors.New("sku option not listed in listing")
			}
			if _, ok := variantMap[selection.Variant]; !ok {
				return errors.New("sku variant not listed in option")
			}
		}
	}
	if len(sl.Listing.Item.Price) > SentenceMaxCharacters {
		return coreiface.ErrTooManyCharacters{"item.price", strconv.Itoa(SentenceMaxCharacters)}
	}
	_, ok := new(big.Int).SetString(sl.Listing.Item.Price, 10)
	if !ok {
		return errors.New("invalid item price")
	}

	// Taxes
	if len(sl.Listing.Taxes) > MaxListItems {
		return coreiface.ErrTooManyItems{"taxes", strconv.Itoa(MaxListItems)}
	}
	for _, tax := range sl.Listing.Taxes {
		if tax.TaxType == "" {
			return coreiface.ErrMissingField("taxes.taxtype")
		}
		if len(tax.TaxType) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"taxes.taxtype", strconv.Itoa(WordMaxCharacters)}
		}
		if len(tax.TaxRegions) == 0 {
			return errors.New("tax must specify at least one region")
		}
		if len(tax.TaxRegions) > MaxCountryCodes {
			return fmt.Errorf("number of tax regions is greater than the max of %d", MaxCountryCodes)
		}
		if tax.Percentage == 0 || tax.Percentage > 100 {
			return errors.New("tax percentage must be between 0 and 100")
		}
	}

	// Coupons
	if len(sl.Listing.Coupons) > MaxListItems {
		return coreiface.ErrTooManyItems{"coupons", strconv.Itoa(MaxListItems)}
	}
	for _, coupon := range sl.Listing.Coupons {
		if len(coupon.Title) > CouponTitleMaxCharacters {
			return coreiface.ErrTooManyCharacters{"coupons.title", strconv.Itoa(SentenceMaxCharacters)}
		}
		if len(coupon.GetDiscountCode()) > CodeMaxCharacters {
			return coreiface.ErrTooManyCharacters{"coupons.discountcode", strconv.Itoa(CodeMaxCharacters)}
		}
		if coupon.GetPercentDiscount() > 100 {
			return errors.New("percent discount cannot be over 100 percent")
		}
		n, _ := new(big.Int).SetString(sl.Listing.Item.Price, 10)
		discountVal := coupon.GetPriceDiscount()
		flag := false
		if discountVal != "" {
			if len(discountVal) > SentenceMaxCharacters {
				return coreiface.ErrTooManyCharacters{"coupons.pricediscount", strconv.Itoa(SentenceMaxCharacters)}
			}
			discount0, ok := new(big.Int).SetString(discountVal, 10)
			if !ok {
				return errors.New("invalid price discount")
			}
			if n.Cmp(discount0) < 0 {
				return errors.New("price discount cannot be greater than the item price")
			}
			if discount0.Cmp(big.NewInt(0)) == 0 {
				flag = true
			}
		}
		if coupon.GetPercentDiscount() == 0 && flag {
			return errors.New("coupons must have at least one positive discount value")
		}
	}

	// Moderators
	if len(sl.Listing.Moderators) > MaxListItems {
		return coreiface.ErrTooManyItems{"moderators", strconv.Itoa(MaxListItems)}
	}
	for _, moderator := range sl.Listing.Moderators {
		_, err := peer.Decode(moderator)
		if err != nil {
			return errors.New("moderator IDs must be valid")
		}
	}

	// TermsAndConditions
	if len(sl.Listing.TermsAndConditions) > PolicyMaxCharacters {
		return coreiface.ErrTooManyCharacters{"termsandconditions", strconv.Itoa(PolicyMaxCharacters)}
	}

	// RefundPolicy
	if len(sl.Listing.RefundPolicy) > PolicyMaxCharacters {
		return coreiface.ErrTooManyCharacters{"refundpolicy", strconv.Itoa(PolicyMaxCharacters)}
	}

	// Type-specific validations
	if sl.Listing.Metadata.ContractType == pb.Listing_Metadata_PHYSICAL_GOOD {
		err := validatePhysicalListing(sl.Listing)
		if err != nil {
			return err
		}
	} else if sl.Listing.Metadata.ContractType == pb.Listing_Metadata_CRYPTOCURRENCY {
		err := n.validateCryptocurrencyListing(sl.Listing)
		if err != nil {
			return err
		}
	}

	// Format-specific validations
	if sl.Listing.Metadata.Format == pb.Listing_Metadata_MARKET_PRICE {
		err := validateMarketPriceListing(sl.Listing)
		if err != nil {
			return err
		}
	}

	// Validate vendor ID
	if sl.Listing.VendorID == nil {
		return coreiface.ErrMissingField("vendorID")
	}
	if len(sl.Listing.VendorID.Handle) > SentenceMaxCharacters {
		return coreiface.ErrTooManyCharacters{"vendorID.handle", strconv.Itoa(SentenceMaxCharacters)}
	}
	if sl.Listing.VendorID.Pubkeys == nil {
		return coreiface.ErrMissingField("vendorID.pubkeys")
	}
	identityPubkey, err := crypto.UnmarshalPublicKey(sl.Listing.VendorID.Pubkeys.Identity)
	if err != nil {
		return errors.New("invalid vendor identity public key")
	}
	peerID, err := peer.IDFromPublicKey(identityPubkey)
	if err != nil {
		return fmt.Errorf("%w: %s", coreiface.ErrInternalServer, err)
	}
	if peerID.Pretty() != sl.Listing.VendorID.PeerID {
		return errors.New("vendor peerID does not match public key")
	}
	if len(sl.Listing.VendorID.Pubkeys.Escrow) != 33 {
		return errors.New("vendor escrow pubkey invalid length")
	}
	ecPubkey, err := btcec.ParsePubKey(sl.Listing.VendorID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return errors.New("invalid vendor escrow public key")
	}
	sig, err := btcec.ParseSignature(sl.Listing.VendorID.Sig, btcec.S256())
	if err != nil {
		return errors.New("invalid vendor identity signature")
	}
	idHash := sha256.Sum256([]byte(sl.Listing.VendorID.PeerID))
	valid := sig.Verify(idHash[:], ecPubkey)
	if !valid {
		return errors.New("invalid secp256k1 signature on vendor identity key")
	}

	// Validate signature on listing
	ser, err := proto.Marshal(sl.Listing)
	if err != nil {
		return fmt.Errorf("%w: %s", coreiface.ErrInternalServer, err)
	}
	valid, err = identityPubkey.Verify(ser, sl.Signature)
	if err != nil {
		return fmt.Errorf("%w: %s", coreiface.ErrInternalServer, err)
	}
	if !valid {
		return errors.New("invalid signature on listing")
	}

	return nil
}

// deserializeAndValidate accepts a byte slice of a serialized SignedListing
// and deserializes and validates it.
func (n *OpenBazaarNode) deserializeAndValidate(listingBytes []byte, cid cid.Cid) (*pb.SignedListing, error) {
	signedListing := new(pb.SignedListing)
	if err := jsonpb.UnmarshalString(string(listingBytes), signedListing); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	if err := n.validateListing(signedListing); err != nil {
		return nil, fmt.Errorf("%w: %s", coreiface.ErrNotFound, err)
	}
	signedListing.Cid = cid.String()
	return signedListing, nil
}

// validatePhysicalListing validates the part of the listing that is relevant to
// physical listings.
func validatePhysicalListing(listing *pb.Listing) error {
	if len(listing.Item.Condition) > SentenceMaxCharacters {
		return coreiface.ErrTooManyCharacters{"item.condition", strconv.Itoa(SentenceMaxCharacters)}
	}
	if len(listing.Item.Options) > MaxListItems {
		return fmt.Errorf("number of options is greater than the max of %d", MaxListItems)
	}

	// ShippingOptions
	if len(listing.ShippingOptions) == 0 {
		return coreiface.ErrMissingField("shippingoptions")
	}
	if len(listing.ShippingOptions) > MaxListItems {
		return fmt.Errorf("number of shipping options is greater than the max of %d", MaxListItems)
	}
	var shippingTitles []string
	for _, shippingOption := range listing.ShippingOptions {
		if shippingOption.Name == "" {
			return coreiface.ErrMissingField("shippingoptions.name")
		}
		if len(shippingOption.Name) > WordMaxCharacters {
			return coreiface.ErrTooManyCharacters{"shippingoptions.name", strconv.Itoa(WordMaxCharacters)}
		}
		for _, t := range shippingTitles {
			if t == shippingOption.Name {
				return errors.New("shipping option titles must be unique")
			}
		}
		shippingTitles = append(shippingTitles, shippingOption.Name)
		if shippingOption.Type > pb.Listing_ShippingOption_FIXED_PRICE {
			return errors.New("unknown shipping option type")
		}
		if len(shippingOption.Regions) == 0 {
			return coreiface.ErrMissingField("shippingoptions.regions")
		}
		if err := validShippingRegion(shippingOption); err != nil {
			return fmt.Errorf("invalid shipping option (%s): %s", shippingOption.String(), err.Error())
		}
		if len(shippingOption.Regions) > MaxCountryCodes {
			return fmt.Errorf("number of shipping regions is greater than the max of %d", MaxCountryCodes)
		}
		if len(shippingOption.Services) == 0 && shippingOption.Type != pb.Listing_ShippingOption_LOCAL_PICKUP {
			return errors.New("at least one service must be specified for a shipping option when not local pickup")
		}
		if len(shippingOption.Services) > MaxListItems {
			return fmt.Errorf("number of shipping services is greater than the max of %d", MaxListItems)
		}
		var serviceTitles []string
		for _, option := range shippingOption.Services {
			if option.Name == "" {
				return coreiface.ErrMissingField("shippingoptions.services.name")
			}
			if len(option.Name) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"shippingoptions.services.name", strconv.Itoa(WordMaxCharacters)}
			}
			for _, t := range serviceTitles {
				if t == option.Name {
					return errors.New("shipping option services names must be unique")
				}
			}
			serviceTitles = append(serviceTitles, option.Name)
			if option.EstimatedDelivery == "" {
				return coreiface.ErrMissingField("shippingoptions.services.estimateddelivery")
			}
			if len(option.EstimatedDelivery) > SentenceMaxCharacters {
				return coreiface.ErrTooManyCharacters{"shippingoptions.services.estimateddelivery", strconv.Itoa(SentenceMaxCharacters)}
			}
			if len(option.Price) > WordMaxCharacters {
				return coreiface.ErrTooManyCharacters{"shippingoptions.services.price", strconv.Itoa(WordMaxCharacters)}
			}
		}
	}

	return nil
}

// validateCryptocurrencyListing validates the part of the listing that is relevant to
// cryptocurrency listings.
func (n *OpenBazaarNode) validateCryptocurrencyListing(listing *pb.Listing) error {
	switch {
	case len(listing.Coupons) > 0:
		return coreiface.ErrCryptocurrencyListingIllegalField("coupons")
	case len(listing.Item.Options) > 0:
		return coreiface.ErrCryptocurrencyListingIllegalField("item.options")
	case len(listing.ShippingOptions) > 0:
		return coreiface.ErrCryptocurrencyListingIllegalField("shippingOptions")
	case len(listing.Item.Condition) > 0:
		return coreiface.ErrCryptocurrencyListingIllegalField("item.condition")
	}

	return nil
}

// validateMarketPriceListing validates the part of the listing that is relevant to
// market price cryptocurrency listings.
func validateMarketPriceListing(listing *pb.Listing) error {
	if listing.Item.Price != "" {
		n, _ := new(big.Int).SetString(listing.Item.Price, 10)
		if n.Cmp(big.NewInt(0)) > 0 {
			return coreiface.ErrMarketPriceListingIllegalField("item.price")
		}
	}

	if listing.Item.CryptoListingPriceModifier != 0 {
		listing.Item.CryptoListingPriceModifier = float32(int(listing.Item.CryptoListingPriceModifier*100.0)) / 100.0
	}

	if listing.Item.CryptoListingPriceModifier < PriceModifierMin ||
		listing.Item.CryptoListingPriceModifier > PriceModifierMax {
		return coreiface.ErrPriceModifierOutOfRange{
			Min: PriceModifierMin,
			Max: PriceModifierMax,
		}
	}

	return nil
}

// validShippingRegion checks that the shipping region is in our list
// of counties in the proto file.
func validShippingRegion(shippingOption *pb.Listing_ShippingOption) error {
	for _, region := range shippingOption.Regions {
		if int32(region) == 0 {
			return coreiface.ErrMissingField("shippingoptions.regions")
		}
		_, ok := proto.EnumValueMap("CountryCode")[region.String()]
		if !ok {
			return errors.New("shipping region undefined")
		}
		if ok {
			if int32(region) > 500 {
				return errors.New("shipping region must not be continent")
			}
		}
	}
	return nil
}

// swapCouponHashesWithDiscountCodes swaps a listing's coupon hashes for the underlying
// discount code (the hash preimage). We do this for our own listings before sending them
// out of the API so that API consumers can see the discount code for our own listings.
func swapCouponHashesWithDiscountCodes(listing *pb.SignedListing, coupons []models.Coupon) *pb.SignedListing {
	couponMap := make(map[string]string)
	for _, coupon := range coupons {
		couponMap[coupon.Hash] = coupon.Code
	}
	for i, listingCoupon := range listing.Listing.Coupons {
		code, ok := couponMap[listingCoupon.GetHash()]
		if ok {
			listing.Listing.Coupons[i].Code = &pb.Listing_Coupon_DiscountCode{DiscountCode: code}
		}
	}
	return listing
}

package utils

import (
	"crypto/sha256"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// ValidateRating returns an error if the rating is invalid, otherwise nil.
func ValidateRating(rating *pb.Rating) error {
	if rating.VendorID == nil || rating.VendorID.Pubkeys == nil {
		return errors.New("invalid vendor ID")
	}

	if rating.VendorSig == nil || rating.VendorSig.RatingKey == nil {
		return errors.New("invalid vendor signature")
	}

	if rating.Overall < 1 || rating.Overall > 5 {
		return errors.New("overall rating out of range")
	}
	if rating.Quality < 1 || rating.Quality > 5 {
		return errors.New("quality rating out of range")
	}
	if rating.Description < 1 || rating.Description > 5 {
		return errors.New("description rating out of range")
	}
	if rating.DeliverySpeed < 1 || rating.DeliverySpeed > 5 {
		return errors.New("delivery speed rating out of range")
	}
	if rating.CustomerService < 1 || rating.CustomerService > 5 {
		return errors.New("customer service rating out of range")
	}
	if len(rating.Review) > 10000 {
		return errors.New("review greater than max characters")
	}

	// Validate the vendor's signature
	vendorKey, err := crypto.UnmarshalPublicKey(rating.VendorID.Pubkeys.Identity)
	if err != nil {
		return err
	}

	cpy := proto.Clone(rating.VendorSig)
	cpy.(*pb.RatingSignature).VendorSignature = nil
	ser, err := proto.Marshal(cpy)
	if err != nil {
		return err
	}
	valid, err := vendorKey.Verify(ser, rating.VendorSig.VendorSignature)
	if !valid || err != nil {
		return errors.New("invalid vendor signature")
	}

	// Validate vendor peerID matches pubkey
	id, err := peer.Decode(rating.VendorID.PeerID)
	if err != nil {
		return err
	}
	if !id.MatchesPublicKey(vendorKey) {
		return errors.New("vendor ID does not match public key")
	}

	// Validate buyer signature if not anonymous
	if rating.BuyerID != nil {
		if rating.BuyerID.Pubkeys == nil {
			return errors.New("buyer public key is nil")
		}
		buyerKey, err := crypto.UnmarshalPublicKey(rating.BuyerID.Pubkeys.Identity)
		if err != nil {
			return err
		}
		valid, err = buyerKey.Verify(rating.VendorSig.RatingKey, rating.BuyerSig)
		if !valid || err != nil {
			return errors.New("invalid buyer signature")
		}

		// Validate buyer peerID matches pubkey
		id, err := peer.Decode(rating.BuyerID.PeerID)
		if err != nil {
			return err
		}
		if !id.MatchesPublicKey(buyerKey) {
			return errors.New("buyer ID does not match public key")
		}
	}

	// Validate rating signature
	cpy = proto.Clone(rating)
	cpy.(*pb.Rating).RatingSignature = nil
	ser, err = proto.Marshal(cpy)
	if err != nil {
		return err
	}
	ratingKey, err := btcec.ParsePubKey(rating.VendorSig.RatingKey, btcec.S256())
	if err != nil {
		return err
	}
	sig, err := btcec.ParseSignature(rating.RatingSignature, btcec.S256())
	if err != nil {
		return err
	}
	hashed := sha256.Sum256(ser)
	valid = sig.Verify(hashed[:], ratingKey)
	if !valid {
		return errors.New("invalid rating signature")
	}

	return nil
}

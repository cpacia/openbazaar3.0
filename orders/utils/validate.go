package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
	"time"
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
		ratingSigHash := sha256.Sum256(rating.VendorSig.RatingKey)
		valid, err = buyerKey.Verify(ratingSigHash[:], rating.BuyerSig)
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

// ValidateBuyerID validates the ID is well formed and contains a valid signature.
func ValidateBuyerID(id *pb.ID) error {
	if id.Pubkeys == nil {
		return errors.New("buyer id pubkeys is nil")
	}
	idPubkey, err := crypto.UnmarshalPublicKey(id.Pubkeys.Identity)
	if err != nil {
		return fmt.Errorf("invalid buyer ID pubkey: %s", err)
	}
	pid, err := peer.IDFromPublicKey(idPubkey)
	if err != nil {
		return fmt.Errorf("invalid buyer ID pubkey: %s", err)
	}
	if pid.Pretty() != id.PeerID {
		return errors.New("buyer ID does not match pubkey")
	}
	escrowPubkey, err := btcec.ParsePubKey(id.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return errors.New("invalid buyer escrow pubkey")
	}
	sig, err := btcec.ParseSignature(id.Sig, btcec.S256())
	if err != nil {
		return errors.New("invalid buyer ID signature")
	}
	idHash := sha256.Sum256([]byte(id.PeerID))
	valid := sig.Verify(idHash[:], escrowPubkey)
	if !valid {
		return errors.New("invalid buyer ID signature")
	}
	return nil
}

func ValidatePayment(order *pb.OrderOpen, escrowTimeoutHours uint32, wal iwallet.Wallet) error {
	if order.Payment.Amount == "" {
		return errors.New("payment amount not set")
	}
	if ok := validateBigString(order.Payment.Amount); !ok {
		return errors.New("payment amount not valid")
	}
	if order.Payment.Address == "" {
		return errors.New("order payment address is empty")
	}
	chaincode, err := hex.DecodeString(order.Payment.Chaincode)
	if err != nil {
		return fmt.Errorf("chaincode parse error: %s", err)
	}
	vendorEscrowPubkey, err := btcec.ParsePubKey(order.Listings[0].Listing.VendorID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return err
	}
	vendorKey, err := GenerateEscrowPublicKey(vendorEscrowPubkey, chaincode)
	if err != nil {
		return err
	}
	buyerEscrowPubkey, err := btcec.ParsePubKey(order.BuyerID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return err
	}
	buyerKey, err := GenerateEscrowPublicKey(buyerEscrowPubkey, chaincode)
	if err != nil {
		return err
	}
	if order.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		_, err := peer.Decode(order.Payment.Moderator)
		if err != nil {
			return errors.New("invalid moderator selection")
		}
		moderatorEscrowPubkey, err := btcec.ParsePubKey(order.Payment.ModeratorKey, btcec.S256())
		if err != nil {
			return err
		}
		moderatorKey, err := GenerateEscrowPublicKey(moderatorEscrowPubkey, chaincode)
		if err != nil {
			return err
		}

		escrowTimeoutWallet, walletSupportsEscrowTimeout := wal.(iwallet.EscrowWithTimeout)
		if !walletSupportsEscrowTimeout {
			escrowTimeoutHours = 0
		}
		var (
			address iwallet.Address
			script  []byte
		)
		if escrowTimeoutHours > 0 {
			timeout := time.Hour * time.Duration(escrowTimeoutHours)
			address, script, err = escrowTimeoutWallet.CreateMultisigWithTimeout([]btcec.PublicKey{*buyerKey, *vendorKey, *moderatorKey}, 2, timeout, *vendorKey)
			if err != nil {
				return err
			}
		} else {
			escrowWallet, ok := wal.(iwallet.Escrow)
			if !ok {
				return errors.New("wallet does not support escrow")
			}
			address, script, err = escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey, *moderatorKey}, 2)
			if err != nil {
				return err
			}
		}

		if order.Payment.Address != address.String() {
			return errors.New("invalid moderated payment address")
		}
		if order.Payment.Script != hex.EncodeToString(script) {
			return errors.New("invalid moderated payment script")
		}
	} else if order.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
		escrowWallet, ok := wal.(iwallet.Escrow)
		if !ok {
			return errors.New("wallet does not support escrow")
		}
		address, script, err := escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey}, 1)
		if err != nil {
			return err
		}
		if order.Payment.Address != address.String() {
			return errors.New("invalid cancelable payment address")
		}
		if order.Payment.Script != hex.EncodeToString(script) {
			return errors.New("invalid cancelable payment script")
		}
	} else if order.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		return errors.New("invalid payment method")
	}
	if order.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		if order.Payment.EscrowReleaseFee == "" {
			return errors.New("escrow release fee is empty")
		}
		if ok := validateBigString(order.Payment.EscrowReleaseFee); !ok {
			return errors.New("escrow release fee not valid")
		}
	}
	return nil
}

// validateBigString validates that the string is a base10 big number.
func validateBigString(s string) bool {
	_, ok := new(big.Int).SetString(s, 10)
	return ok
}

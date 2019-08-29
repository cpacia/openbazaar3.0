package core

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
)

// generateRatingPublicKeys uses the chaincode from the order to deterministically generate public keys
// from our rating public key. This allows us to recover the private key to sign the rating with later on
// as long as we know the chaincode.
//
// This is a hardened key so anyone who views the key should not be able to derive our master pubkey key
// nor any other keys we use for ratings.
func generateRatingPublicKeys(ratingPubKey *btcec.PublicKey, numKeys int, chaincode []byte) ([][]byte, error) {
	hdKey := hdkeychain.NewExtendedKey(
		chaincfg.MainNetParams.HDPublicKeyID[:],
		ratingPubKey.SerializeCompressed(),
		chaincode,
		[]byte{0x00, 0x00, 0x00, 0x00},
		0,
		0,
		false)

	return generateHdKeys(hdKey, numKeys, false)
}

// generateRatingPrivateKeys does the same thing as it's public key counterpart except it
// returns the private keys.
func generateRatingPrivateKeys(ratingPrivKey *btcec.PrivateKey, numKeys int, chaincode []byte) ([][]byte, error) {
	hdKey := hdkeychain.NewExtendedKey(
		chaincfg.MainNetParams.HDPrivateKeyID[:],
		ratingPrivKey.Serialize(),
		chaincode,
		[]byte{0x00, 0x00, 0x00, 0x00},
		0,
		0,
		false)

	return generateHdKeys(hdKey, numKeys, true)
}

// generateEscrowPublicKey uses the chaincode from the order to deterministically generate public keys
// from our escrow public key. This allows us to recover the private key to sign the transaction with later
// on as long as we know the chaincode.
//
// This is a hardened key so anyone who views the key should not be able to derive our master pubkey key
// nor any other keys we use for orders.
func generateEscrowPublicKey(escrowPubKey *btcec.PublicKey, chaincode []byte) (*btcec.PublicKey, error) {
	hdKey := hdkeychain.NewExtendedKey(
		chaincfg.MainNetParams.HDPublicKeyID[:],
		escrowPubKey.SerializeCompressed(),
		chaincode,
		[]byte{0x00, 0x00, 0x00, 0x00},
		0,
		0,
		false)

	key, err := generateHdKey(hdKey)
	if err != nil {
		return nil, err
	}
	return key.ECPubKey()
}

// generateEscrowPrivateKey does the same thing as it's public key counterpart except it
// returns the private keys.
func generateEscrowPrivateKey(escrowPrivKey *btcec.PrivateKey, chaincode []byte) (*btcec.PrivateKey, error) {
	hdKey := hdkeychain.NewExtendedKey(
		chaincfg.MainNetParams.HDPrivateKeyID[:],
		escrowPrivKey.Serialize(),
		chaincode,
		[]byte{0x00, 0x00, 0x00, 0x00},
		0,
		0,
		false)

	key, err := generateHdKey(hdKey)
	if err != nil {
		return nil, err
	}
	return key.ECPrivKey()
}

// generateHdKey returns a single child key from the provided hd key.
func generateHdKey(hdKey *hdkeychain.ExtendedKey) (*hdkeychain.ExtendedKey, error) {
	i := 0
	for {
		childKey, err := hdKey.Child(hdkeychain.HardenedKeyStart + uint32(i))
		if err != nil {
			// Small chance this can fail due to weird curve stuff.
			// Bip32 spec calls for skipping to next key.
			i++
			continue
		}
		return childKey, nil
	}
}

// generateHdKeys is a helper function that can generate from either public or private
// keys.
func generateHdKeys(hdKey *hdkeychain.ExtendedKey, numKeys int, priv bool) ([][]byte, error) {
	keys := make([][]byte, 0, numKeys)
	i := 0
	for len(keys) < numKeys {
		childKey, err := hdKey.Child(hdkeychain.HardenedKeyStart + uint32(i))
		if err != nil {
			// Small chance this can fail due to weird curve stuff.
			// Bip32 spec calls for skipping to next key.
			i++
			continue
		}
		if priv {
			priv, err := childKey.ECPrivKey()
			if err != nil {
				return nil, err
			}
			keys = append(keys, priv.Serialize())
		} else {
			pub, err := childKey.ECPubKey()
			if err != nil {
				return nil, err
			}
			keys = append(keys, pub.SerializeCompressed())
		}

		i++
	}
	return keys, nil
}

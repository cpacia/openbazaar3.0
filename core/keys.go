package core

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
)

// generateRatingPublicKeys uses the chaincode from the order to deterministically generated public keys
// from our rating public key. This allows us to recover the private key to sign the rating with later on
// as long as we know the order timestamp.
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

	return generateHdKeys(hdKey, numKeys)
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

	return generateHdKeys(hdKey, numKeys)
}

// generateHdKeys is a helper function that can generate from either public or private
// keys.
func generateHdKeys(hdKey *hdkeychain.ExtendedKey, numKeys int) ([][]byte, error) {
	keys := make([][]byte, 0, numKeys)
	i := 0
	for len(keys) < numKeys {
		childPriv, err := hdKey.Child(hdkeychain.HardenedKeyStart + uint32(i))
		if err != nil {
			// Small chance this can fail due to weird curve stuff.
			// Bip32 spec calls for skipping to next key.
			continue
		}
		pub, err := childPriv.ECPubKey()
		if err != nil {
			return nil, err
		}
		keys = append(keys, pub.SerializeCompressed())
		i++
	}
	return keys, nil
}

package net

import (
	"github.com/cpacia/openbazaar3.0/net/pb"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"testing"
)

func TestEncryptCurve25519(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "Hello World!!!"
	ciphertext, err := Encrypt(pub, &pb.ChatMessage{Message: plaintext})
	if err != nil {
		t.Fatal(err)
	}
	decrypted := new(pb.ChatMessage)
	err = Decrypt(priv, ciphertext, decrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted.Message != plaintext {
		t.Errorf("Expected plaintext of %s, got %s", plaintext, decrypted.Message)
	}
}

package utils

import (
	"crypto/rand"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"testing"
)

func TestSignOrderMessage(t *testing.T) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	orderOpen := pb.OrderOpen{
		AlternateContactInfo: "1234",
	}

	a, err := ptypes.MarshalAny(&orderOpen)
	if err != nil {
		t.Fatal(err)
	}

	order := npb.OrderMessage{
		Message:     a,
		MessageType: npb.OrderMessage_ORDER_OPEN,
		OrderID:     "abc",
	}

	err = SignOrderMessage(&order, priv)
	if err != nil {
		t.Fatal(err)
	}

	cpy := proto.Clone(&order)
	cpy.(*npb.OrderMessage).Signature = nil

	ser, err := proto.Marshal(cpy)
	if err != nil {
		t.Fatal(err)
	}

	valid, err := pub.Verify(ser, order.Signature)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Error("invalid signature")
	}
}

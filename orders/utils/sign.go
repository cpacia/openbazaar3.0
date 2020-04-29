package utils

import (
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
)

// SignOrderMessage puts a signature on an order message using the IPFS private
// key. The protobuf serialization of the message object without the signature
// is what is signed.
func SignOrderMessage(message *pb.OrderMessage, privKey crypto.PrivKey) error {
	message.Signature = nil
	ser, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	sig, err := privKey.Sign(ser)
	if err != nil {
		return err
	}

	message.Signature = sig
	return nil
}

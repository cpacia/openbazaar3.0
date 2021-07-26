package utils

import (
	"crypto/sha256"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/multiformats/go-multihash"
)

// MultihashSha256 hashes the data with sha256 and returns a multihash representation.
func MultihashSha256(b []byte) (*multihash.Multihash, error) {
	h := sha256.Sum256(b)
	encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return nil, err
	}
	multihash, err := multihash.Cast(encoded)
	if err != nil {
		return nil, err
	}
	return &multihash, err
}

func HashListing(sl *pb.SignedListing) (*multihash.Multihash, error) {
	ser, err := proto.Marshal(sl)
	if err != nil {
		return nil, err
	}
	return MultihashSha256(ser)
}

func CalcOrderID(orderOpen *pb.OrderOpen) (*multihash.Multihash, error) {
	ser, err := proto.Marshal(orderOpen)
	if err != nil {
		return nil, err
	}
	return MultihashSha256(ser)
}

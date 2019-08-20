package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multihash"
)

func NewOrder() (*pb.OrderOpen, *crypto.PrivKey, error) {
	privKeyBytes, err := hex.DecodeString("080112406e22f498c42014ea4485c2d4bdffd90fb3c4ee394f0aaa49a61a7b4e51235e016efc82dba17659db9daf4c8d1e39818f0d41ce9919876e299f56c71031375944")
	if err != nil {
		return nil, nil, err
	}
	privkey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	pubkeyBytes, err := privkey.GetPublic().Bytes()
	if err != nil {
		return nil, nil, err
	}
	escrowPubkey, err := hex.DecodeString("029f4ead6d340076a4060df48d723714541d5a23c8547e50414a22d6e360ea1b3b")
	if err != nil {
		return nil, nil, err
	}
	idSig, err := hex.DecodeString("3045022100a24d967b45f058dfcb2b32a58cd2434c38af677741bd1607e54cf9fad3df869f02202758cf2a0f9c85efaf88488488692e0bb3862b634d0fa1ca357dda287ef3c55c")
	if err != nil {
		return nil, nil, err
	}
	ratingKey, err := hex.DecodeString("02fcaa2903a6aeff06eb5660d82cf3cd6ce686e7d2e2c23a12b23ea0cbbaf04e99")
	if err != nil {
		return nil, nil, err
	}

	listing := NewSignedListing()
	ser, err := proto.Marshal(listing)
	if err != nil {
		return nil, nil, err
	}
	h := sha256.Sum256(ser)
	encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return nil, nil, err
	}
	listingHash, err := multihash.Cast(encoded)
	if err != nil {
		return nil, nil, err
	}

	order := &pb.OrderOpen{
		Listings: []*pb.SignedListing{
			listing,
		},
		RefundAddress: "01ce26dc69094af9246ea7e7ce9970aff2b81cc9",
		Shipping: &pb.OrderOpen_Shipping{
			ShipTo:       "Peter Griffin",
			Address:      "31 Spooner Street",
			City:         "Quahog",
			State:        "RI",
			PostalCode:   "90210",
			Country:      pb.CountryCode_UNITED_STATES,
			AddressNotes: "Don't leave in on the porch. Cleveland steals my packages.",
		},
		BuyerID: &pb.ID{
			PeerID: "12D3KooWDLv1VhTC7N7aasWkeG1Hyv3QgATYJNrbosnzEykMSxhg",
			Handle: "@assman",
			Pubkeys: &pb.ID_Pubkeys{
				Identity: pubkeyBytes,
				Escrow:   escrowPubkey,
			},
			Sig: idSig,
		},
		Timestamp: ptypes.TimestampNow(),
		Items: []*pb.OrderOpen_Item{
			{
				ListingHash: listingHash.B58String(),
				Quantity:    1,
				Options: []*pb.OrderOpen_Item_Option{
					{
						Name:  "size",
						Value: "large",
					},
					{
						Name:  "color",
						Value: "red",
					},
				},
				ShippingOption: &pb.OrderOpen_Item_ShippingOption{
					Name:    "usps",
					Service: "standard",
				},
			},
		},
		Payment: &pb.OrderOpen_Payment{
			Method:  pb.OrderOpen_Payment_DIRECT,
			Amount:  "466454170",
			Address: "068885193265b763a7510377a61176192622be07",
			Coin:    "TMCK",
		},
		RatingKeys:           [][]byte{ratingKey},
		AlternateContactInfo: "peter@familyguy.net",
		Version:              1,
	}

	ser, err = proto.Marshal(order)
	if err != nil {
		return nil, nil, err
	}
	sig, err := privkey.Sign(ser)
	if err != nil {
		return nil, nil, err
	}
	order.Signature = sig

	return order, &privkey, nil
}

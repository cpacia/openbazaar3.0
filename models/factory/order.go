package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multihash"
)

func NewOrder() (*pb.OrderOpen, crypto.PrivKey, error) {
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

	pid, err := peer.IDFromPublicKey(privkey.GetPublic())
	if err != nil {
		return nil, nil, err
	}

	escrowPrivkeyBytes, err := hex.DecodeString("e93fc130413a742e96844ac2d2b38b380081b0a54ddc3aac4e5bdaecb598ff38")
	if err != nil {
		return nil, nil, err
	}
	escrowPrivkey, escrowPubkey := btcec.PrivKeyFromBytes(btcec.S256(), escrowPrivkeyBytes)

	sigHash := sha256.Sum256([]byte(pid.Pretty()))
	sig, err := escrowPrivkey.Sign(sigHash[:])
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
			PeerID: pid.Pretty(),
			Handle: "@assman",
			Pubkeys: &pb.ID_Pubkeys{
				Identity: pubkeyBytes,
				Escrow:   escrowPubkey.SerializeCompressed(),
			},
			Sig: sig.Serialize(),
		},
		Timestamp: ptypes.TimestampNow(),
		Items: []*pb.OrderOpen_Item{
			{
				ListingHash: listingHash.B58String(),
				Quantity:    "1",
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
			Method:           pb.OrderOpen_Payment_CANCELABLE,
			Amount:           "4992221",
			Address:          "d2a09570b1275d9f865cf290f5aca9f514d1ac4e5b770fdc0b59153059a8c09c",
			Coin:             "TMCK",
			EscrowReleaseFee: "10",
			Script:           "036d60859d9a78554a69e15cf6044c7c3d81744038048719e87cdbe3ab5d159f100221ec8c291efd5f39d100590d5ec2038655ea1ece51de656255a1f7567fd731d900000001",
		},
		RatingKeys:           [][]byte{ratingKey},
		AlternateContactInfo: "peter@familyguy.net",
	}

	ser, err = proto.Marshal(order)
	if err != nil {
		return nil, nil, err
	}
	orderSig, err := privkey.Sign(ser)
	if err != nil {
		return nil, nil, err
	}
	order.Signature = orderSig

	return order, privkey, nil
}

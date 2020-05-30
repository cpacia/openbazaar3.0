package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"github.com/OpenBazaar/jsonpb"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"testing"
	"time"
)

func TestOpenBazaarNode_Ratings(t *testing.T) {
	mockNet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer mockNet.TearDown()

	rating, err := newTestRating()
	if err != nil {
		t.Fatal(err)
	}
	if err := utils.ValidateRating(rating); err != nil {
		t.Fatal(err)
	}

	var id cid.Cid
	err = mockNet.Nodes()[0].repo.DB().Update(func(tx database.Tx) error {
		if err := tx.SetRating(rating); err != nil {
			return err
		}

		m := jsonpb.Marshaler{Indent: "    "}
		out, err := m.MarshalToString(rating)
		if err != nil {
			return err
		}

		id, err = mockNet.Nodes()[0].cid([]byte(out))
		if err != nil {
			return err
		}

		var index models.RatingIndex
		if err := index.AddRating(rating, id); err != nil {
			return err
		}
		return tx.SetRatingIndex(index)
	})
	if err != nil {
		t.Fatal(err)
	}

	ratings, err := mockNet.Nodes()[0].GetMyRatings()
	if err != nil {
		t.Fatal(err)
	}
	if len(ratings) != 1 {
		t.Errorf("Expected 1 rating, got %d", len(ratings))
	}
	if ratings[0].Count != 1 {
		t.Errorf("Expected 1 rating count, got %d", ratings[0].Count)
	}
	if ratings[0].Ratings[0] != id.String() {
		t.Errorf("Expected cid %s, got %s", id, ratings[0].Ratings[0])
	}

	done := make(chan struct{})
	mockNet.Nodes()[0].Publish(done)
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("timed out while publishing")
	}

	ratings, err = mockNet.Nodes()[1].GetRatings(context.Background(), mockNet.Nodes()[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ratings) != 1 {
		t.Errorf("Expected 1 rating, got %d", len(ratings))
	}
	if ratings[0].Count != 1 {
		t.Errorf("Expected 1 rating count, got %d", ratings[0].Count)
	}
	if ratings[0].Ratings[0] != id.String() {
		t.Errorf("Expected cid %s, got %s", id, ratings[0].Ratings[0])
	}

	rating2, err := mockNet.Nodes()[1].GetRating(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	if rating2.Review != rating.Review {
		t.Errorf("Expected review %s, got %s", rating.Review, rating2.Review)
	}
}

func newTestRating() (*pb.Rating, error) {
	vendorPrivkey, vendorPubkey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	vendorPubkeyBytes, err := vendorPubkey.Bytes()
	if err != nil {
		return nil, err
	}
	vendorRatingKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}
	vendorID, err := peer.IDFromPublicKey(vendorPubkey)
	if err != nil {
		return nil, err
	}
	idHash := sha256.Sum256([]byte(vendorID.Pretty()))
	vendorIDSig, err := vendorRatingKey.Sign(idHash[:])
	if err != nil {
		return nil, err
	}

	buyerPrivkey, buyerPubkey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	buyerPubkeyBytes, err := buyerPubkey.Bytes()
	if err != nil {
		return nil, err
	}
	buyerRatingKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}
	buyerID, err := peer.IDFromPublicKey(buyerPubkey)
	if err != nil {
		return nil, err
	}
	idHash = sha256.Sum256(buyerRatingKey.PubKey().SerializeCompressed())
	buyerIDSig, err := buyerPrivkey.Sign(idHash[:])
	if err != nil {
		return nil, err
	}

	r := &pb.RatingSignature{
		Slug:      "slug",
		RatingKey: buyerRatingKey.PubKey().SerializeCompressed(),
	}

	ser, err := proto.Marshal(r)
	if err != nil {
		return nil, err
	}

	sig, err := vendorPrivkey.Sign(ser)
	if err != nil {
		return nil, err
	}
	r.VendorSignature = sig

	rating := &pb.Rating{
		Timestamp: ptypes.TimestampNow(),

		VendorSig: r,
		VendorID: &pb.ID{
			PeerID: vendorID.Pretty(),
			Handle: "@handle",
			Pubkeys: &pb.ID_Pubkeys{
				Identity: vendorPubkeyBytes,
				Escrow:   vendorRatingKey.PubKey().SerializeCompressed(),
			},
			Sig: vendorIDSig.Serialize(),
		},
		BuyerID: &pb.ID{
			PeerID: buyerID.Pretty(),
			Handle: "@handle",
			Pubkeys: &pb.ID_Pubkeys{
				Identity: buyerPubkeyBytes,
				Escrow:   buyerRatingKey.PubKey().SerializeCompressed(),
			},
			Sig: vendorIDSig.Serialize(),
		},
		BuyerSig: buyerIDSig,

		Overall:         uint32(5),
		Quality:         uint32(4),
		CustomerService: uint32(3),
		Description:     uint32(2),
		DeliverySpeed:   uint32(1),
		Review:          "excellent",

		BuyerName: "Bob",
	}

	ser, err = proto.Marshal(rating)
	if err != nil {
		return nil, err
	}

	hashed := sha256.Sum256(ser)

	ratingSig, err := buyerRatingKey.Sign(hashed[:])
	if err != nil {
		return nil, err
	}
	rating.RatingSignature = ratingSig.Serialize()

	return rating, nil
}

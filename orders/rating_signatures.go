package orders

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func (op *OrderProcessor) processRatingSignaturesMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	rs := new(pb.RatingSignatures)
	if err := ptypes.UnmarshalAny(message.Message, rs); err != nil {
		return nil, err
	}

	dup, err := isDuplicate(rs, order.SerializedRatingSignatures)
	if err != nil {
		return nil, err
	}
	if order.SerializedRatingSignatures != nil && !dup {
		log.Errorf("Duplicate RATING_SIGNATURES message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	if len(rs.Sigs) != len(orderOpen.RatingKeys) {
		return nil, errors.New("vendor sent incorrect number of rating signatures")
	}

	pub, err := crypto.UnmarshalPublicKey(orderOpen.Listings[0].Listing.VendorID.Pubkeys.Identity)
	if err != nil {
		return nil, err
	}

	for i, sig := range rs.Sigs {
		listing, err := utils.ExtractListing(orderOpen.Items[i].ListingHash, orderOpen.Listings)
		if err != nil {
			return nil, err
		}

		if sig.Slug != listing.Slug {
			return nil, errors.New("rating signature contains incorrect slug")
		}

		cpy := proto.Clone(sig)
		cpy.(*pb.RatingSignature).VendorSignature = nil

		ser, err := proto.Marshal(cpy)
		if err != nil {
			return nil, err
		}

		valid, err := pub.Verify(ser, sig.VendorSignature)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, errors.New("invalid vendor signature on rating key")
		}
	}

	log.Infof("Received RATING_SIGNATURES message for order %s", order.ID)

	event := &events.RatingSignaturesReceived{
		OrderID: order.ID.String(),
	}
	return event, order.PutMessage(message)
}

// sendRatingSignatures signs the buyer's rating keys and sends the signatures to the buyer. We want to do
// this right after the order is funded.
func (op *OrderProcessor) sendRatingSignatures(dbtx database.Tx, order *models.Order, orderOpen *pb.OrderOpen) error {
	rs := &pb.RatingSignatures{
		Timestamp: ptypes.TimestampNow(),
	}
	for i, item := range orderOpen.Items {
		listing, err := utils.ExtractListing(item.ListingHash, orderOpen.Listings)
		if err != nil {
			return err
		}

		r := &pb.RatingSignature{
			Slug:      listing.Slug,
			RatingKey: orderOpen.RatingKeys[i],
		}

		ser, err := proto.Marshal(r)
		if err != nil {
			return err
		}

		sig, err := op.identityPrivateKey.Sign(ser)
		if err != nil {
			return err
		}
		r.VendorSignature = sig

		rs.Sigs = append(rs.Sigs, r)
	}

	rsAny, err := ptypes.MarshalAny(rs)
	if err != nil {
		return err
	}

	om := npb.OrderMessage{
		OrderID:     order.ID.String(),
		MessageType: npb.OrderMessage_RATING_SIGNATURES,
		Message:     rsAny,
	}

	if err := utils.SignOrderMessage(&om, op.identityPrivateKey); err != nil {
		return err
	}

	payload, err := ptypes.MarshalAny(&om)
	if err != nil {
		return err
	}

	messageID := make([]byte, 20)
	if _, err := rand.Read(messageID); err != nil {
		return err
	}

	message := npb.Message{
		MessageType: npb.Message_ORDER,
		MessageID:   hex.EncodeToString(messageID),
		Payload:     payload,
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}

	if err := op.messenger.ReliablySendMessage(dbtx, buyer, &message, nil); err != nil {
		return err
	}

	return order.PutMessage(&om)
}

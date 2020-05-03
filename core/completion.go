package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/crypto"
)

// CompleteOrder builds a OrderComplete message and sends it to the vendor. The ratings slice must
// include one rating per item and must be in the same order as the items in the order. If you wish
// to include the buyerID with the rating then send includingIDInRating.
func (n *OpenBazaarNode) CompleteOrder(orderID models.OrderID, ratings []models.Rating, includeIDInRating bool, done chan struct{}) error {
	var (
		order   models.Order
		profile *models.Profile
		err     error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		profile, err = tx.GetProfile()
		if err != nil {
			return err
		}
		return tx.Read().Where("id = ?", orderID.String()).Find(&order).Error
	})
	if err != nil {
		return err
	}

	if !order.CanComplete(n.Identity()) {
		return fmt.Errorf("%w: order is not in a state where it can be completed", coreiface.ErrBadRequest)
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return err
	}

	fulfillments, err := order.OrderFulfillmentMessages()
	if err != nil {
		return err
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	ratingSignatures, err := order.RatingSignaturesMessage()
	if err != nil {
		return err
	}

	if len(ratings) != len(orderOpen.Items) {
		return errors.New("number of ratings does not equal number of items in the order")
	}

	if len(ratingSignatures.Sigs) != len(orderOpen.Items) {
		return errors.New("missing rating signatures from vendor needed to build rating")
	}

	chaincode, err := hex.DecodeString(orderOpen.Payment.Chaincode)
	if err != nil {
		return err
	}

	ratingKeys, err := utils.GenerateRatingPrivateKeys(n.ratingMasterKey, len(orderOpen.Items), chaincode)
	if err != nil {
		return err
	}

	completeMsg := &pb.OrderComplete{
		Timestamp: ptypes.TimestampNow(),
	}

	for i, rating := range ratings {
		ratingPB := &pb.Rating{
			Timestamp: ptypes.TimestampNow(),

			VendorSig: ratingSignatures.Sigs[i],
			VendorID:  orderOpen.Listings[0].Listing.VendorID,

			Overall:         uint32(rating.Overall),
			Quality:         uint32(rating.Quality),
			CustomerService: uint32(rating.CustomerService),
			Description:     uint32(rating.Description),
			DeliverySpeed:   uint32(rating.DeliverySpeed),
			Review:          rating.Review,
		}

		if includeIDInRating {
			identityPubkey, err := crypto.MarshalPublicKey(n.ipfsNode.PrivateKey.GetPublic())
			if err != nil {
				return err
			}

			idHash := sha256.Sum256([]byte(n.Identity().Pretty()))
			sig, err := n.escrowMasterKey.Sign(idHash[:])
			if err != nil {
				return err
			}

			ratingPB.BuyerName = profile.Name
			ratingPB.BuyerID = &pb.ID{
				PeerID: n.Identity().Pretty(),
				Pubkeys: &pb.ID_Pubkeys{
					Identity: identityPubkey,
					Escrow:   n.escrowMasterKey.PubKey().SerializeCompressed(),
				},
				Handle: profile.Handle,
				Sig:    sig.Serialize(),
			}

			buyerSig, err := n.ipfsNode.PrivateKey.Sign(ratingPB.VendorSig.RatingKey)
			if err != nil {
				return err
			}
			ratingPB.BuyerSig = buyerSig
		}

		ser, err := proto.Marshal(ratingPB)
		if err != nil {
			return err
		}

		hashed := sha256.Sum256(ser)

		ratingSig, err := ratingKeys[i].Sign(hashed[:])
		if err != nil {
			return err
		}
		ratingPB.RatingSignature = ratingSig.Serialize()

		completeMsg.Ratings = append(completeMsg.Ratings, ratingPB)
	}

	if orderOpen.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		wallet, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
		if err != nil {
			return err
		}

		release, err := n.buildEscrowRelease(&order, wallet,
			iwallet.NewAddress(fulfillments[0].ReleaseInfo.ToAddress, iwallet.CoinType(orderOpen.Payment.Coin)),
			iwallet.NewAmount(fulfillments[0].ReleaseInfo.TransactionFee))
		if err != nil {
			return err
		}

		completeMsg.ReleaseInfo = release
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		completeAny, err := ptypes.MarshalAny(completeMsg)
		if err != nil {
			return err
		}

		m := &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_ORDER_COMPLETE,
			Message:     completeAny,
		}

		if err := utils.SignOrderMessage(m, n.ipfsNode.PrivateKey); err != nil {
			return err
		}

		payload, err := ptypes.MarshalAny(m)
		if err != nil {
			return err
		}

		message := newMessageWithID()
		message.MessageType = npb.Message_ORDER
		message.Payload = payload

		_, err = n.orderProcessor.ProcessMessage(tx, buyer, m)
		if err != nil {
			return err
		}

		return n.messenger.ReliablySendMessage(tx, vendor, message, done)
	})
}

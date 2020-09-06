package orders

import (
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
	"os"
)

func (op *OrderProcessor) processOrderCompleteMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	complete := new(pb.OrderComplete)
	if err := ptypes.UnmarshalAny(message.Message, complete); err != nil {
		return nil, err
	}

	dup, err := isDuplicate(complete, order.SerializedOrderComplete)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderComplete != nil && !dup {
		log.Errorf("Duplicate ORDER_COMPLETE message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	if order.SerializedOrderCancel != nil {
		log.Errorf("Received ORDER_COMPLETE message for order %s after ORDER_CANCEL", order.ID)
		return nil, ErrUnexpectedMessage
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	if len(complete.Ratings) != len(orderOpen.Items) {
		return nil, errors.New("number of ratings does not equal number of items in the order")
	}

	if len(complete.Ratings) > 0 {
		for _, rating := range complete.Ratings {
			if err := utils.ValidateRating(rating); err != nil {
				return nil, err
			}
		}
	}
	if order.Role() == models.RoleVendor && len(complete.Ratings) > 0 {
		index, err := dbtx.GetRatingIndex()
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		for _, rating := range complete.Ratings {
			m := jsonpb.Marshaler{Indent: "    "}
			out, err := m.MarshalToString(rating)
			if err != nil {
				return nil, err
			}

			id, err := op.calcCIDFunc([]byte(out))
			if err != nil {
				return nil, err
			}
			err = index.AddRating(rating, id)
			if err != nil {
				return nil, err
			}
			if err := dbtx.SetRatingIndex(index); err != nil {
				return nil, err
			}
			if err := dbtx.SetRating(rating); err != nil {
				return nil, err
			}
		}
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	if order.Role() == models.RoleVendor && complete.GetReleaseInfo() != nil && orderOpen.Payment.Method == pb.OrderOpen_Payment_MODERATED && order.SerializedDisputeOpen == nil {
		fulfillments, err := order.OrderFulfillmentMessages()
		if err != nil {
			return nil, err
		}
		if err := op.releaseCompleteEscrowFunds(wallet, orderOpen, fulfillments, complete.GetReleaseInfo()); err != nil {
			log.Errorf("Error releasing funds from escrow during order complete processing: %s", err.Error())
		}
	}

	if order.Role() == models.RoleVendor {
		log.Infof("Received ORDER_COMPLETE message for order %s", order.ID)
	} else if order.Role() == models.RoleBuyer {
		log.Infof("Processed own ORDER_COMPLETE for order %s", order.ID)
	}

	event := &events.OrderCompletion{
		OrderID: order.ID.String(),
		Thumbnail: events.Thumbnail{
			Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
			Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
		},
		BuyerHandle: orderOpen.BuyerID.Handle,
		BuyerID:     orderOpen.BuyerID.PeerID,
	}
	return event, order.PutMessage(message)
}

func (op *OrderProcessor) releaseCompleteEscrowFunds(wallet iwallet.Wallet, orderOpen *pb.OrderOpen, fulfillments []*pb.OrderFulfillment, releaseInfo *pb.EscrowRelease) error {
	escrowWallet, ok := wallet.(iwallet.Escrow)
	if !ok {
		return errors.New("wallet for moderated order does not support escrow")
	}

	payoutAddr := fulfillments[0].ReleaseInfo.ToAddress

	if releaseInfo.ToAddress != payoutAddr {
		return errors.New("escrow release does not pay out to expected vendor address")
	}
	_, ok = new(big.Int).SetString(releaseInfo.ToAmount, 10)
	if !ok {
		return errors.New("invalid payment amount")
	}
	txn := iwallet.Transaction{
		To: []iwallet.SpendInfo{
			{
				Address: iwallet.NewAddress(releaseInfo.ToAddress, iwallet.CoinType(orderOpen.Payment.Coin)),
				Amount:  iwallet.NewAmount(releaseInfo.ToAmount),
			},
		},
	}

	for _, id := range releaseInfo.FromIDs {
		txn.From = append(txn.From, iwallet.SpendInfo{ID: id})
	}

	var buyerSigs []iwallet.EscrowSignature
	for _, sig := range releaseInfo.EscrowSignatures {
		buyerSigs = append(buyerSigs, iwallet.EscrowSignature{
			Index:     int(sig.Index),
			Signature: sig.Signature,
		})
	}

	script, err := hex.DecodeString(orderOpen.Payment.Script)
	if err != nil {
		return err
	}

	fulfillmentSigs := fulfillments[0].ReleaseInfo.EscrowSignatures
	vendorSigs := make([]iwallet.EscrowSignature, 0, len(fulfillmentSigs))
	for _, sig := range fulfillmentSigs {
		vendorSigs = append(vendorSigs, iwallet.EscrowSignature{
			Index:     int(sig.Index),
			Signature: sig.Signature,
		})
	}

	wtx, err := wallet.Begin()
	if err != nil {
		return err
	}
	if _, err := escrowWallet.BuildAndSend(wtx, txn, [][]iwallet.EscrowSignature{buyerSigs, vendorSigs}, script); err != nil {
		return err
	}

	return wtx.Commit()
}

package orders

import (
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
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

	if err := order.PutMessage(message); err != nil {
		if models.IsDuplicateTransactionError(err) {
			return nil, nil
		}
		return nil, err
	}

	if complete.Rating != nil {
		if err := utils.ValidateRating(complete.Rating); err != nil {
			return nil, err
		}
	}
	if order.Role() == models.RoleVendor && complete.Rating != nil {
		err = op.db.Update(func(tx database.Tx) error {
			index, err := tx.GetRatingIndex()
			if err != nil {
				return err
			}
			m := jsonpb.Marshaler{Indent: "    "}
			out, err := m.MarshalToString(complete.Rating)
			if err != nil {
				return err
			}

			id, err := op.calcCIDFunc([]byte(out))
			if err != nil {
				return err
			}
			err = index.AddRating(complete.Rating, id)
			if err != nil {
				return err
			}
			if err := tx.SetRatingIndex(index); err != nil {
				return err
			}
			return tx.SetRating(complete.Rating)
		})
		if err != nil {
			return nil, err
		}
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	if order.Role() == models.RoleVendor && complete.GetReleaseInfo() != nil && orderOpen.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		if err := op.releaseEscrowFunds(wallet, orderOpen, complete.GetReleaseInfo()); err != nil {
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
		BuyerHandle: orderOpen.Listings[0].Listing.VendorID.Handle,
		BuyerID:     orderOpen.Listings[0].Listing.VendorID.PeerID,
	}
	return event, nil
}

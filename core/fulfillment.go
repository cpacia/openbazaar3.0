package core

import (
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
)

// FulfillOrder sends an order fulfillment to the remote peer and updates the order state.
func (n *OpenBazaarNode) FulfillOrder(orderID models.OrderID, fulfillments []models.Fulfillment, done chan struct{}) error {
	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Find(&order).Error
	})
	if err != nil {
		return err
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		return err
	}

	fulfillmentMsg := &pb.OrderFulfillment{
		Timestamp: ptypes.TimestampNow(),
	}

	for _, f := range fulfillments {
		if f.ItemIndex > len(orderOpen.Items) {
			return fmt.Errorf("%w: invalid item index", coreiface.ErrBadRequest)
		}

		item := &pb.OrderFulfillment_FulfilledItem{
			Note:      f.Note,
			ItemIndex: uint32(f.ItemIndex),
		}
		if f.PhysicalDelivery != nil {
			item.Delivery = &pb.OrderFulfillment_FulfilledItem_PhysicalDelivery_{
				PhysicalDelivery: &pb.OrderFulfillment_FulfilledItem_PhysicalDelivery{
					Shipper:        f.PhysicalDelivery.Shipper,
					TrackingNumber: f.PhysicalDelivery.TrackingNumber,
				},
			}
		} else if f.DigitalDelivery != nil {
			item.Delivery = &pb.OrderFulfillment_FulfilledItem_DigitalDelivery_{
				DigitalDelivery: &pb.OrderFulfillment_FulfilledItem_DigitalDelivery{
					Url:      f.DigitalDelivery.URL,
					Password: f.DigitalDelivery.Password,
				},
			}
		} else if f.CryptocurrencyDelivery != nil {
			item.Delivery = &pb.OrderFulfillment_FulfilledItem_CryptocurrencyDelivery_{
				CryptocurrencyDelivery: &pb.OrderFulfillment_FulfilledItem_CryptocurrencyDelivery{
					TransactionID: f.CryptocurrencyDelivery.TransactionID,
				},
			}
		} else {
			return fmt.Errorf("%w: a delivery option must be selected", coreiface.ErrBadRequest)
		}
		fulfillmentMsg.Fulfillments = append(fulfillmentMsg.Fulfillments, item)
	}

	if !order.CanFulfill(n.Identity()) {
		return fmt.Errorf("%w: order is not in a state where it can be fulfilled", coreiface.ErrBadRequest)
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		if orderOpen.Payment.Method == pb.OrderOpen_Payment_MODERATED {
			wallet, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
			if err != nil {
				return err
			}
			addr, err := wallet.NewAddress()
			if err != nil {
				return err
			}
			escrowWallet, ok := wallet.(iwallet.Escrow)
			if !ok {
				return errors.New("wallet does not support escrow")
			}
			fee, err := escrowWallet.EstimateEscrowFee(2, iwallet.FlNormal)
			if err != nil {
				return err
			}
			release, err := n.buildEscrowRelease(&order, wallet, addr, fee)
			if err != nil {
				return err
			}
			fulfillmentMsg.ReleaseInfo = release
		}

		fulfillmentAny, err := ptypes.MarshalAny(fulfillmentMsg)
		if err != nil {
			return err
		}

		resp := &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_ORDER_FULFILLMENT,
			Message:     fulfillmentAny,
		}

		if err := utils.SignOrderMessage(resp, n.ipfsNode.PrivateKey); err != nil {
			return err
		}

		payload, err := ptypes.MarshalAny(resp)
		if err != nil {
			return err
		}

		message := newMessageWithID()
		message.MessageType = npb.Message_ORDER
		message.Payload = payload

		_, err = n.orderProcessor.ProcessMessage(tx, vendor, resp)
		if err != nil {
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, buyer, message, done); err != nil {
			return err
		}

		return nil
	})
}

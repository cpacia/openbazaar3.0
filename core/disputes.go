package core

import (
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/peer"
	"gorm.io/gorm"
)

// OpenDispute sends a disputeOpen message to both the moderator and the other party to the order and
// updates the order state.
func (n *OpenBazaarNode) OpenDispute(orderID models.OrderID, reason string, done chan struct{}) error {
	done1, done2 := make(chan struct{}), make(chan struct{})
	go func() {
		if done != nil {
			<-done1
			<-done2
			close(done)
		}
	}()

	var order models.Order
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", orderID.String()).Find(&order).Error
	})
	if err != nil {
		return err
	}

	if !order.CanDispute() {
		return fmt.Errorf("%w: order is not in a state where it can be disputed", coreiface.ErrBadRequest)
	}

	buyer, err := order.Buyer()
	if err != nil {
		return err
	}
	vendor, err := order.Vendor()
	if err != nil {
		return err
	}

	moderator, err := order.Moderator()
	if err != nil {
		return err
	}

	var (
		role = pb.DisputeOpen_BUYER
		to   = vendor
		from = buyer
	)
	if order.Role() == models.RoleVendor {
		role = pb.DisputeOpen_VENDOR
		to = buyer
		from = vendor
	}

	serializedContract, err := order.MarshalBinary()
	if err != nil {
		return err
	}

	disputeOpen := &pb.DisputeOpen{
		Timestamp: ptypes.TimestampNow(),
		OpenedBy:  role,
		Reason:    reason,
		Contract:  serializedContract,
	}

	return n.repo.DB().Update(func(tx database.Tx) error {
		disputeOpenAny, err := ptypes.MarshalAny(disputeOpen)
		if err != nil {
			return err
		}

		m := &npb.OrderMessage{
			OrderID:     order.ID.String(),
			MessageType: npb.OrderMessage_DISPUTE_OPEN,
			Message:     disputeOpenAny,
		}

		if err := utils.SignOrderMessage(m, n.ipfsNode.PrivateKey); err != nil {
			return err
		}

		payload, err := ptypes.MarshalAny(m)
		if err != nil {
			return err
		}

		message1 := newMessageWithID()
		message1.MessageType = npb.Message_ORDER
		message1.Payload = payload

		_, err = n.orderProcessor.ProcessMessage(tx, from, m)
		if err != nil {
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, to, message1, done1); err != nil {
			return err
		}

		message2 := newMessageWithID()
		message2.MessageType = npb.Message_DISPUTE
		message2.Payload = payload

		return n.messenger.ReliablySendMessage(tx, moderator, message2, done2)
	})
}

// handleOrderMessage is the handler for the ORDER message. It sends it off to the order
// order processor for processing.
func (n *OpenBazaarNode) handleDisputeMessage(from peer.ID, message *npb.Message) error {
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	if message.MessageType != npb.Message_DISPUTE {
		return errors.New("message is not type DISPUTE")
	}

	order := new(npb.OrderMessage)
	if err := ptypes.UnmarshalAny(message.Payload, order); err != nil {
		return err
	}

	switch order.MessageType {
	case npb.OrderMessage_DISPUTE_OPEN:
		disputeOpen := new(pb.DisputeOpen)
		if err := ptypes.UnmarshalAny(order.Message, disputeOpen); err != nil {
			return err
		}

		orderOpen, err := extractOrderOpen(disputeOpen.Contract)
		if err != nil {
			return err
		}

		var (
			role           = models.RoleBuyer
			disputer       = orderOpen.BuyerID.PeerID
			disputerHandle = orderOpen.BuyerID.Handle
			disputee       = orderOpen.Listings[0].Listing.VendorID.PeerID
			disputeeHandle = orderOpen.Listings[0].Listing.VendorID.Handle
		)
		if disputeOpen.OpenedBy == pb.DisputeOpen_VENDOR {
			role = models.RoleVendor
			disputer = orderOpen.Listings[0].Listing.VendorID.PeerID
			disputerHandle = orderOpen.Listings[0].Listing.VendorID.Handle
			disputee = orderOpen.BuyerID.PeerID
			disputeeHandle = orderOpen.BuyerID.Handle
		}

		validationErrors, err := n.validateDisputeOpen(from, disputeOpen)
		if err != nil {
			return err
		}

		return n.repo.DB().Update(func(dbtx database.Tx) error {
			dbtx.RegisterCommitHook(func() {
				n.eventBus.Emit(&events.CaseOpen{
					CaseID:         order.OrderID,
					DisputerID:     disputer,
					DisputerHandle: disputerHandle,
					DisputeeID:     disputee,
					DisputeeHandle: disputeeHandle,
					Thumbnail: events.Thumbnail{
						Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
						Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
					},
				})
				log.Infof("Received new case. ID: %s", order.OrderID)
			})

			var disputeCase models.Case
			err := dbtx.Read().Where("id = ?", order.OrderID).First(&disputeCase).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			disputeCase.ID = models.OrderID(order.OrderID)
			if err := disputeCase.PutValidationErrors(validationErrors, role); err != nil {
				return err
			}

			if disputeCase.SerializedDisputeOpen != nil {
				return fmt.Errorf("received duplicate DISPUTE_OPEN message from %s", from.Pretty())
			}

			err = disputeCase.PutDisputeOpen(disputeOpen)
			if err != nil {
				return err
			}
			return dbtx.Save(&disputeCase)
		})
	case npb.OrderMessage_DISPUTE_UPDATE:
		disputeUpdate := new(pb.DisputeUpdate)
		if err := ptypes.UnmarshalAny(order.Message, disputeUpdate); err != nil {
			return err
		}

		orderOpen, err := extractOrderOpen(disputeUpdate.Contract)
		if err != nil {
			return err
		}

		var (
			disputer       = orderOpen.BuyerID.PeerID
			disputerHandle = orderOpen.BuyerID.Handle
			disputee       = orderOpen.Listings[0].Listing.VendorID.PeerID
			disputeeHandle = orderOpen.Listings[0].Listing.VendorID.Handle
		)
		if orderOpen.BuyerID.PeerID == from.Pretty() {
			disputer = orderOpen.Listings[0].Listing.VendorID.PeerID
			disputerHandle = orderOpen.Listings[0].Listing.VendorID.Handle
			disputee = orderOpen.BuyerID.PeerID
			disputeeHandle = orderOpen.BuyerID.Handle
		}

		return n.repo.DB().Update(func(dbtx database.Tx) error {
			dbtx.RegisterCommitHook(func() {
				n.eventBus.Emit(&events.CaseUpdate{
					CaseID:         order.OrderID,
					DisputerID:     disputer,
					DisputerHandle: disputerHandle,
					DisputeeID:     disputee,
					DisputeeHandle: disputeeHandle,
					Thumbnail: events.Thumbnail{
						Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
						Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
					},
				})
				log.Infof("Received case update for case %s", order.OrderID)
			})

			var disputeCase models.Case
			err := dbtx.Read().Where("id = ?", order.OrderID).First(&disputeCase).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			disputeCase.ID = models.OrderID(order.OrderID)

			// TODO: validate dispute update contract

			err = disputeCase.PutDisputeUpdate(disputeUpdate)
			if err != nil {
				return err
			}
			return dbtx.Save(&disputeCase)
		})
	}
	return nil
}

func (n *OpenBazaarNode) validateDisputeOpen(from peer.ID, dispute *pb.DisputeOpen) (validationErrors []error, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = fmt.Errorf("dispute contract missing required field: %s", x)
			case error:
				err = fmt.Errorf("dispute contract missing required field: %w", x)
			default:
				err = errors.New("unknown dispute open validation panic")
			}
		}
	}()

	orderOpen, err := extractOrderOpen(dispute.Contract)
	if err != nil {
		return nil, err
	}

	openedByPeer := orderOpen.BuyerID.PeerID
	if dispute.OpenedBy == pb.DisputeOpen_VENDOR {
		openedByPeer = orderOpen.Listings[0].Listing.VendorID.PeerID
	}

	if openedByPeer != from.Pretty() {
		return nil, errors.New("dispute open openedBy peerID does not match peer that sent the message")
	}

	if orderOpen.Payment.Moderator != n.Identity().Pretty() {
		return nil, errors.New("selected moderator does not match own peerID")
	}

	if orderOpen.Payment.Method != pb.OrderOpen_Payment_MODERATED {
		return nil, errors.New("order payment method is not type moderated")
	}

	wal, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, fmt.Errorf("cannot validate order. coin not supported by moderator. %w", err)
	}

	for i, listing := range orderOpen.Listings {
		err := n.validateListing(listing)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("listing %d in contract is invalid: %s", i, err.Error()))
		}
	}

	var escrowTimeoutHours uint32
	for i, item := range orderOpen.Items {
		listing, err := utils.ExtractListing(item.ListingHash, orderOpen.Listings)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("order does not contain any listings that match the listing ID for item %d", i))
			continue
		}

		if listing.Metadata.EscrowTimeoutHours > escrowTimeoutHours {
			escrowTimeoutHours = listing.Metadata.EscrowTimeoutHours
		}
	}

	if err := utils.ValidateBuyerID(orderOpen.BuyerID); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("invalid buyer ID in order: %s", err.Error()))
	}

	if err := utils.ValidatePayment(orderOpen, escrowTimeoutHours, wal); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("order payment is invalid: %s", err.Error()))
	}

	return validationErrors, nil
}

func extractOrderOpen(contract []byte) (*pb.OrderOpen, error) {
	var c pb.Contract
	if err := proto.Unmarshal(contract, &c); err != nil {
		return nil, err
	}
	return c.OrderOpen, nil
}

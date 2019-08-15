package orders

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multihash"
)

func (op *OrderProcessor) handleOrderOpenMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	dup, err := isDuplicate(message, order.SerializedOrderOpen)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderOpen != nil && !dup {
		log.Error("Duplicate ORDER_OPEN message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	orderOpen := new(pb.OrderOpen)
	if err := ptypes.UnmarshalAny(message.Message, orderOpen); err != nil {
		return nil, err
	}

	var validationError bool
	// If the validation fails and we are the vendor, we send a REJECT message back
	// to the buyer. The reject message also gets saved with this order.
	if err := op.validateOrderOpen(dbtx, orderOpen); err != nil {
		log.Error("ORDER_OPEN message for order %s from %s failed to validate: %s", order.ID, orderOpen.BuyerID.PeerID, err)
		if op.identity != peer {
			reject := pb.OrderReject{
				Type:   pb.OrderReject_VALIDATION_ERROR,
				Reason: err.Error(),
			}

			rejectAny, err := ptypes.MarshalAny(&reject)
			if err != nil {
				return nil, err
			}

			resp := npb.OrderMessage{
				OrderID:     order.ID.String(),
				MessageType: npb.OrderMessage_ORDER_REJECT,
				Message:     rejectAny,
			}

			payload, err := ptypes.MarshalAny(&resp)
			if err != nil {
				return nil, err
			}

			messageID := make([]byte, 20)
			if _, err := rand.Read(messageID); err != nil {
				return nil, err
			}

			message := npb.Message{
				MessageType: npb.Message_ORDER,
				MessageID:   hex.EncodeToString(messageID),
				Payload:     payload,
			}

			if err := op.messenger.ReliablySendMessage(dbtx, peer, &message, nil); err != nil {
				return nil, err
			}

			if err := order.PutMessage(&resp); err != nil {
				return nil, err
			}
		}
		validationError = true
	}

	var event interface{}
	// TODO: do we want to emit an event in the case of a validation error?
	if !validationError && op.identity != peer {
		event = &events.OrderNotification{
			ID: order.ID.String(),
		}
	}

	if err := order.PutMessage(orderOpen); err != nil {
		return nil, err
	}

	return event, nil
}

// CalculateOrderTotal calculates and returns the total for the order with all
// the provided options.
func CalculateOrderTotal(order *pb.OrderOpen) (iwallet.Amount, error) {
	// TODO
	return iwallet.NewAmount(0), nil
}

func (op *OrderProcessor) validateOrderOpen(dbtx database.Tx, order *pb.OrderOpen) error {
	// TODO

	if op.identity.Pretty() != order.BuyerID.PeerID { // If we are vendor.
		// Check to make sure we actually have the item for sale.
		for _, listing := range order.Listings {
			myListing, err := dbtx.GetListing(listing.Listing.Slug)
			if err != nil {
				return fmt.Errorf("item %s is not for sale", listing.Listing.Slug)
			}

			// Zero out the inventory on each listing. We will check
			// inventory later.
			for i := range myListing.Listing.Item.Skus {
				myListing.Listing.Item.Skus[i].Quantity = 0
			}
			for i := range listing.Listing.Item.Skus {
				listing.Listing.Item.Skus[i].Quantity = 0
			}

			// We can tell if we have the listing for sale if the serialized bytes match
			// after we've zeroed out the inventory.
			mySer, err := proto.Marshal(myListing.Listing)
			if err != nil {
				return err
			}

			theirSer, err := proto.Marshal(listing.Listing)
			if err != nil {
				return err
			}

			if !bytes.Equal(mySer, theirSer) {
				return fmt.Errorf("item %s is not for sale", listing.Listing.Slug)
			}
		}
	}

	// Let's check to make sure there is a listing for each
	// item in the order.
	listingHashes := make(map[string]bool)
	for _, listing := range order.Listings {
		ser, err := proto.Marshal(listing)
		if err != nil {
			return err
		}
		h := sha256.Sum256(ser)
		encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
		if err != nil {
			return err
		}
		hash, err := multihash.Cast(encoded)
		if err != nil {
			return err
		}
		listingHashes[hash.B58String()] = true
	}

	for _, item := range order.Items {
		if !listingHashes[item.ListingHash] {
			return fmt.Errorf("listing not found in order for item %s", item.ListingHash)
		}
	}

	return nil
}

package orders

import (
	"crypto/rand"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"reflect"
	"testing"
)

func TestOrderProcessor_processOrderRejectMessage(t *testing.T) {
	op, err := newMockOrderProcessor()
	if err != nil {
		t.Fatal(err)
	}

	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	remotePeer, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	rejectMsg := &pb.OrderReject{
		Type:   pb.OrderReject_VALIDATION_ERROR,
		Reason: "Test",
	}

	rejectAny, err := ptypes.MarshalAny(rejectMsg)
	if err != nil {
		t.Fatal(err)
	}

	orderMsg := &npb.OrderMessage{
		OrderID:     "1234",
		MessageType: npb.OrderMessage_ORDER_REJECT,
		Message:     rejectAny,
	}

	var (
		vendorPeerID   = "xyz"
		vendorHandle   = "abc"
		smallImageHash = "aaaa"
		tinyImageHash  = "bbbb"
	)
	orderOpen := &pb.OrderOpen{
		Listings: []*pb.SignedListing{
			{
				Listing: &pb.Listing{
					VendorID: &pb.ID{
						PeerID: vendorPeerID,
						Handle: vendorHandle,
					},
					Item: &pb.Listing_Item{
						Images: []*pb.Listing_Item_Image{
							{
								Small: smallImageHash,
								Tiny:  tinyImageHash,
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		setup         func(order *models.Order) error
		expectedError error
		expectedEvent interface{}
	}{
		{
			// Normal case where order open exists.
			setup: func(order *models.Order) error {
				order.ID = "1234"
				return order.PutMessage(orderOpen)
			},
			expectedError: nil,
			expectedEvent: &events.OrderDeclinedNotification{
				OrderID: "1234",
				Thumbnail: events.Thumbnail{
					Tiny:  tinyImageHash,
					Small: smallImageHash,
				},
				VendorHandle: vendorHandle,
				VendorID:     vendorPeerID,
			},
		},
		{
			// Order confirmation already exists.
			setup: func(order *models.Order) error {
				order.SerializedOrderReject = nil
				order.SerializedOrderConfirmation = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// Order cancel already exists.
			setup: func(order *models.Order) error {
				order.SerializedOrderReject = nil
				order.SerializedOrderCancel = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// Duplicate order reject.
			setup: func(order *models.Order) error {
				return order.PutMessage(rejectMsg)
			},
			expectedError: nil,
			expectedEvent: nil,
		},
		{
			// Duplicate but different.
			setup: func(order *models.Order) error {
				msg2 := *rejectMsg
				msg2.Type = pb.OrderReject_USER_REJECT
				return order.PutMessage(&msg2)
			},
			expectedError: ErrChangedMessage,
			expectedEvent: nil,
		},
		{
			// Out of order.
			setup: func(order *models.Order) error {
				order.SerializedOrderOpen = nil
				return nil
			},
			expectedError: nil,
			expectedEvent: nil,
		},
	}

	for i, test := range tests {
		order := &models.Order{}
		if err := test.setup(order); err != nil {
			t.Errorf("Test %d setup error: %s", i, err)
			continue
		}
		err := op.db.Update(func(tx database.Tx) error {
			event, err := op.processOrderRejectMessage(tx, order, remotePeer, orderMsg)
			if err != test.expectedError {
				return fmt.Errorf("incorrect error returned. Expected %t, got %t", test.expectedError, err)
			}
			if !reflect.DeepEqual(event, test.expectedEvent) {
				return fmt.Errorf("incorrect event returned")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Error executing db update in test %d: %s", i, err)
		}
	}
}

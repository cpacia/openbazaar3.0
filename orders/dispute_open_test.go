package orders

import (
	"crypto/rand"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"reflect"
	"testing"
)

func TestOrderProcessor_processDisputeOpenMessage(t *testing.T) {
	op, teardown, err := newMockOrderProcessor()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	_, localPub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	localPubkeyBytes, err := crypto.MarshalPublicKey(localPub)
	if err != nil {
		t.Fatal(err)
	}
	localPeer, err := peer.IDFromPublicKey(localPub)
	if err != nil {
		t.Fatal(err)
	}

	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubkeyBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	remotePeer, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	orderID := "1234"

	disputeOpenMsg := &pb.DisputeOpen{
		OpenedBy: pb.DisputeOpen_BUYER,
	}

	disputeOpenAny, err := ptypes.MarshalAny(disputeOpenMsg)
	if err != nil {
		t.Fatal(err)
	}

	orderMsg := &npb.OrderMessage{
		OrderID:     orderID,
		MessageType: npb.OrderMessage_DISPUTE_OPEN,
		Message:     disputeOpenAny,
	}

	var (
		buyerPeerID    = remotePeer.Pretty()
		buyerHandle    = "abc"
		localHandle    = "xyz"
		smallImageHash = "aaaa"
		tinyImageHash  = "bbbb"
	)
	orderOpen := &pb.OrderOpen{
		Listings: []*pb.SignedListing{
			{
				Listing: &pb.Listing{
					Item: &pb.Listing_Item{
						Images: []*pb.Listing_Item_Image{
							{
								Small: smallImageHash,
								Tiny:  tinyImageHash,
							},
						},
					},
					VendorID: &pb.ID{
						PeerID: localPeer.Pretty(),
						Handle: localHandle,
						Pubkeys: &pb.ID_Pubkeys{
							Identity: localPubkeyBytes,
						},
					},
				},
			},
		},
		BuyerID: &pb.ID{
			PeerID: buyerPeerID,
			Handle: buyerHandle,
			Pubkeys: &pb.ID_Pubkeys{
				Identity: pubkeyBytes,
			},
		},
		Payment: &pb.OrderOpen_Payment{
			Coin:      iwallet.CtMock,
			Moderator: "12D3KooWHnpVyu9XDeFoAVayqr9hvc9xPqSSHtCSFLEkKgcz5Wro",
			Method: pb.OrderOpen_Payment_MODERATED,
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
				order.ID = models.OrderID(orderID)

				return order.PutMessage(&npb.OrderMessage{
					Signature:   []byte("abc"),
					Message:     mustBuildAny(orderOpen),
					MessageType: npb.OrderMessage_ORDER_OPEN,
				})
			},
			expectedError: nil,
			expectedEvent: &events.DisputeOpen{
				OrderID: orderID,
				Thumbnail: events.Thumbnail{
					Tiny:  tinyImageHash,
					Small: smallImageHash,
				},
				DisputerID:     buyerPeerID,
				DisputerHandle: buyerHandle,
				DisputeeID:     localPeer.Pretty(),
				DisputeeHandle: localHandle,
			},
		},
		{
			// OrderComplete already exists.
			setup: func(order *models.Order) error {
				order.SerializedOrderComplete = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// PaymentFinalized already exists.
			setup: func(order *models.Order) error {
				order.SerializedPaymentFinalized = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// OrderReject already exists.
			setup: func(order *models.Order) error {
				order.SerializedOrderReject = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// OrderCancel already exists.
			setup: func(order *models.Order) error {
				order.SerializedOrderCancel = []byte{0x00}
				return nil
			},
			expectedError: ErrUnexpectedMessage,
			expectedEvent: nil,
		},
		{
			// Duplicate dispute open.
			setup: func(order *models.Order) error {
				return order.PutMessage(&npb.OrderMessage{
					Signature:   []byte("abc"),
					Message:     disputeOpenAny,
					MessageType: npb.OrderMessage_DISPUTE_OPEN,
				})
			},
			expectedError: nil,
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
			event, err := op.processDisputeOpenMessage(tx, order, remotePeer, orderMsg)
			if err != test.expectedError {
				return fmt.Errorf("incorrect error returned. Expected %t, got %t", test.expectedError, err)
			}
			if !reflect.DeepEqual(event, test.expectedEvent) {
				fmt.Println(event)
				fmt.Println(test.expectedEvent)
				return fmt.Errorf("incorrect event returned")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Error executing db update in test %d: %s", i, err)
		}
	}
}

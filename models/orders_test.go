package models

import (
	"github.com/OpenBazaar/jsonpb"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"testing"
)

func TestOrder_Role(t *testing.T) {
	var order Order

	order.SetRole(RoleVendor)

	ret := order.Role()
	if ret != RoleVendor {
		t.Errorf("Expected RoleVendor, got %d", ret)
	}
}

func TestOrder_ReturnRole(t *testing.T) {
	orderOpen := &pb.OrderOpen{
		Listings: []*pb.SignedListing{
			{
				Listing: &pb.Listing{
					VendorID: &pb.ID{
						PeerID: "QmbN1x5opuJ8FwNyQDaasCRRiTami1WcbNV5oguwHX83g9",
					},
				},
			},
		},
		BuyerID: &pb.ID{
			PeerID: "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
		},
		Payment: &pb.OrderOpen_Payment{
			Moderator: "QmW4cc8jh8vNDza49YVCmFX56tb7QtEGfvcihXEWAKwdcf",
		},
	}
	var order Order
	if err := order.PutMessage(orderOpen); err != nil {
		t.Fatal(err)
	}

	buyerID, err := order.Buyer()
	if err != nil {
		t.Fatal(err)
	}

	if buyerID.Pretty() != orderOpen.BuyerID.PeerID {
		t.Errorf("Incorrect peerID. Expected %s, got %s", orderOpen.BuyerID.PeerID, buyerID.Pretty())
	}

	vendorID, err := order.Vendor()
	if err != nil {
		t.Fatal(err)
	}

	if vendorID.Pretty() != orderOpen.Listings[0].Listing.VendorID.PeerID {
		t.Errorf("Incorrect peerID. Expected %s, got %s", orderOpen.Listings[0].Listing.VendorID.PeerID, vendorID.Pretty())
	}

	moderatorID, err := order.Moderator()
	if err != nil {
		t.Fatal(err)
	}

	if moderatorID.Pretty() != orderOpen.Payment.Moderator {
		t.Errorf("Incorrect peerID. Expected %s, got %s", orderOpen.Payment.Moderator, moderatorID.Pretty())
	}

	orderOpen.Payment.Moderator = ""
	if err := order.PutMessage(orderOpen); err != nil {
		t.Fatal(err)
	}

	_, err = order.Moderator()
	if err == nil {
		t.Error("Expected error from Moderator() method")
	}

}

func TestOrder_Transactions(t *testing.T) {
	var (
		order Order
		id0   = "xyz"
		id1   = "abc"
	)

	err := order.PutTransaction(iwallet.Transaction{
		ID: iwallet.TransactionID(id0),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutTransaction(iwallet.Transaction{
		ID: iwallet.TransactionID(id1),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutTransaction(iwallet.Transaction{
		ID: "abc",
	})
	if err != ErrDuplicateTransaction {
		t.Errorf("Failed to return duplicate transaction error")
	}

	txs, err := order.GetTransactions()
	if err != nil {
		t.Fatal(err)
	}

	for txs[0].ID.String() != id0 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id0, txs[0].ID)
	}
	for txs[1].ID.String() != id1 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id1, txs[1].ID)
	}
}

func TestOrder_PutAndGet(t *testing.T) {
	messages := []proto.Message{
		&pb.OrderOpen{},
		&pb.OrderReject{},
		&pb.OrderCancel{},
		&pb.OrderConfirmation{},
		&pb.OrderFulfillment{},
		&pb.OrderComplete{},
		&pb.DisputeOpen{},
		&pb.DisputeClose{},
		&pb.DisputeUpdate{},
		&pb.Refund{},
		&pb.PaymentFinalized{},
	}

	var order Order
	for i, message := range messages {
		if err := order.PutMessage(message); err != nil {
			t.Errorf("Error putting message %d", i)
		}
	}

	orderOpen, err := order.OrderOpenMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderOpen == nil {
		t.Error("Message is nil")
	}
	orderReject, err := order.OrderRejectMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderReject == nil {
		t.Error("Message is nil")
	}
	orderCancel, err := order.OrderCancelMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderCancel == nil {
		t.Error("Message is nil")
	}
	orderConfirmation, err := order.OrderConfirmationMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderConfirmation == nil {
		t.Error("Message is nil")
	}
	orderFulfillment, err := order.OrderFulfillmentMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderFulfillment == nil {
		t.Error("Message is nil")
	}
	orderComplete, err := order.OrderCompleteMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if orderComplete == nil {
		t.Error("Message is nil")
	}
	disputeOpen, err := order.DisputeOpenMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if disputeOpen == nil {
		t.Error("Message is nil")
	}
	disputeClosed, err := order.DisputeClosedMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if disputeClosed == nil {
		t.Error("Message is nil")
	}
	disputeUpdate, err := order.DisputeUpdateMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if disputeUpdate == nil {
		t.Error("Message is nil")
	}
	refunds, err := order.Refunds()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if refunds == nil {
		t.Error("Message is nil")
	}
	paymentFinalized, err := order.PaymentFinalizedMessage()
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if paymentFinalized == nil {
		t.Error("Message is nil")
	}

	order = Order{}
	orderOpen, err = order.OrderOpenMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	orderReject, err = order.OrderRejectMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	orderCancel, err = order.OrderCancelMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	orderConfirmation, err = order.OrderConfirmationMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	orderFulfillment, err = order.OrderFulfillmentMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	orderComplete, err = order.OrderCompleteMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	disputeOpen, err = order.DisputeOpenMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	disputeClosed, err = order.DisputeClosedMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	disputeUpdate, err = order.DisputeUpdateMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	refunds, err = order.Refunds()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
	paymentFinalized, err = order.PaymentFinalizedMessage()
	if err != ErrMessageDoesNotExist {
		t.Errorf("Get failed to return correct error: %s", err)
	}
}

func TestOrder_Payments(t *testing.T) {
	var (
		order Order
		id0   = "xyz"
		id1   = "abc"
	)

	err := order.PutMessage(&pb.PaymentSent{
		TransactionID: id0,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.PaymentSent{
		TransactionID: id1,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.PaymentSent{
		TransactionID: id0,
	})
	if err != ErrDuplicateTransaction {
		t.Errorf("Failed to return duplicate transaction error")
	}

	payments, err := order.PaymentSentMessages()
	if err != nil {
		t.Fatal(err)
	}

	for payments[0].TransactionID != id0 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id0, payments[0].TransactionID)
	}
	for payments[1].TransactionID != id1 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id1, payments[1].TransactionID)
	}
}

func TestOrder_Refunds(t *testing.T) {
	var (
		order    Order
		id0      = "xyz"
		id1      = "abc"
		release0 = &pb.Refund_ReleaseInfo{ReleaseInfo: &pb.Refund_EscrowRelease{
			EscrowSignatures: []*pb.Refund_Signature{
				{
					From:      []byte{0x00},
					Signature: []byte{0x01},
					Index:     0,
				},
			},
			Address: "abc",
			Amount:  "0",
		}}
		release1 = &pb.Refund_ReleaseInfo{ReleaseInfo: &pb.Refund_EscrowRelease{
			EscrowSignatures: []*pb.Refund_Signature{
				{
					From:      []byte{0x00},
					Signature: []byte{0x02},
					Index:     0,
				},
			},
			Address: "abc",
			Amount:  "1",
		}}
	)

	err := order.PutMessage(&pb.Refund{
		RefundInfo: &pb.Refund_TransactionID{
			TransactionID: id0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.Refund{
		RefundInfo: &pb.Refund_TransactionID{
			TransactionID: id1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.Refund{
		RefundInfo: &pb.Refund_TransactionID{
			TransactionID: id1,
		},
	})
	if err != ErrDuplicateTransaction {
		t.Errorf("Failed to return duplicate transaction error")
	}

	refunds, err := order.Refunds()
	if err != nil {
		t.Fatal(err)
	}

	for refunds[0].GetTransactionID() != id0 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id0, refunds[0].GetTransactionID())
	}
	for refunds[1].GetTransactionID() != id1 {
		t.Errorf("Incorrect txid returned. Expected %s, got %s", id1, refunds[1].GetTransactionID())
	}

	err = order.PutMessage(&pb.Refund{
		RefundInfo: release0,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.Refund{
		RefundInfo: release1,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = order.PutMessage(&pb.Refund{
		RefundInfo: release1,
	})
	if err != ErrDuplicateTransaction {
		t.Errorf("Failed to return duplicate transaction error")
	}

	refunds, err = order.Refunds()
	if err != nil {
		t.Fatal(err)
	}

	marshaler := jsonpb.Marshaler{}

	releaseInfo0, err := marshaler.MarshalToString(&pb.Refund{
		RefundInfo: release0,
	})
	if err != nil {
		t.Fatal(err)
	}

	releaseInfo1, err := marshaler.MarshalToString(&pb.Refund{
		RefundInfo: release1,
	})
	if err != nil {
		t.Fatal(err)
	}

	saved0, err := marshaler.MarshalToString(refunds[2])
	if err != nil {
		t.Fatal(err)
	}

	saved1, err := marshaler.MarshalToString(refunds[3])
	if err != nil {
		t.Fatal(err)
	}

	if releaseInfo0 != saved0 {
		t.Error("Incorrect release info returned")
	}
	if releaseInfo1 != saved1 {
		t.Error("Incorrect release info returned")
	}
}

func TestOrder_ParkedMessages(t *testing.T) {
	var (
		order Order
		msg1  = &npb.OrderMessage{
			MessageType: npb.OrderMessage_ORDER_OPEN,
		}

		msg2 = &npb.OrderMessage{
			MessageType: npb.OrderMessage_ORDER_REJECT,
		}
	)

	msgs, err := order.GetParkedMessages()
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 0 {
		t.Error("Messages should be nil")
	}

	if err := order.ParkMessage(msg1); err != nil {
		t.Fatal(err)
	}
	if err := order.ParkMessage(msg2); err != nil {
		t.Fatal(err)
	}

	msgs, err = order.GetParkedMessages()
	if err != nil {
		t.Fatal(err)
	}

	if msgs[0].MessageType != msg1.MessageType {
		t.Errorf("Expected %s message type got %s", msg1.MessageType.String(), msgs[0].MessageType.String())
	}
	if msgs[1].MessageType != msg2.MessageType {
		t.Errorf("Expected %s message type got %s", msg2.MessageType.String(), msgs[1].MessageType.String())
	}

	if err := order.DeleteParkedMessage(npb.OrderMessage_ORDER_REJECT); err != nil {
		t.Fatal(err)
	}

	msgs, err = order.GetParkedMessages()
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 1 {
		t.Errorf("Failed to delete message")
	}
}

func TestOrder_ErroredMessages(t *testing.T) {
	var (
		order Order
		msg1  = &npb.OrderMessage{
			MessageType: npb.OrderMessage_ORDER_OPEN,
		}

		msg2 = &npb.OrderMessage{
			MessageType: npb.OrderMessage_ORDER_REJECT,
		}
	)

	msgs, err := order.GetErroredMessages()
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 0 {
		t.Error("Messages should be nil")
	}

	if err := order.PutErrorMessage(msg1); err != nil {
		t.Fatal(err)
	}
	if err := order.PutErrorMessage(msg2); err != nil {
		t.Fatal(err)
	}

	msgs, err = order.GetErroredMessages()
	if err != nil {
		t.Fatal(err)
	}

	if msgs[0].MessageType != msg1.MessageType {
		t.Errorf("Expected %s message type got %s", msg1.MessageType.String(), msgs[0].MessageType.String())
	}
	if msgs[1].MessageType != msg2.MessageType {
		t.Errorf("Expected %s message type got %s", msg2.MessageType.String(), msgs[1].MessageType.String())
	}
}

func TestOrder_CanReject(t *testing.T) {
	tests := []struct {
		setup     func(order *Order) error
		ourID     string
		canReject bool
	}{
		{
			// Success
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: true,
		},
		{
			// Nil buyerID
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Is buyer
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Order is nil
			setup: func(order *Order) error {
				return nil
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil reject
			setup: func(order *Order) error {
				order.SerializedOrderReject = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil cancel
			setup: func(order *Order) error {
				order.SerializedOrderCancel = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil confirmation
			setup: func(order *Order) error {
				order.SerializedOrderConfirmation = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil fulfillment
			setup: func(order *Order) error {
				order.SerializedOrderFulfillment = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil complete
			setup: func(order *Order) error {
				order.SerializedOrderComplete = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil dispute open
			setup: func(order *Order) error {
				order.SerializedDisputeOpen = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil dispute close
			setup: func(order *Order) error {
				order.SerializedDisputeClosed = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil dispute update
			setup: func(order *Order) error {
				order.SerializedDisputeUpdate = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil refund
			setup: func(order *Order) error {
				order.SerializedRefunds = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
		{
			// Non nil payment finalized
			setup: func(order *Order) error {
				order.SerializedPaymentFinalized = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:     "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canReject: false,
		},
	}

	for i, test := range tests {
		var order Order
		if err := test.setup(&order); err != nil {
			t.Errorf("Test %d setup failed: %s", i, err)
		}

		pid, err := peer.IDB58Decode(test.ourID)
		if err != nil {
			t.Errorf("Test %d peerID decode error: %s", i, err)
		}

		canReject := order.CanReject(pid)
		if canReject != test.canReject {
			t.Errorf("Got incorrect result. Expected %t, got %t", test.canReject, canReject)
		}
	}
}

func TestOrder_CanConfirm(t *testing.T) {
	tests := []struct {
		setup      func(order *Order) error
		ourID      string
		canConfirm bool
	}{
		{
			// Success
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: true,
		},
		{
			// Nil buyerID
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Is buyer
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Order is nil
			setup: func(order *Order) error {
				return nil
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil reject
			setup: func(order *Order) error {
				order.SerializedOrderReject = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil cancel
			setup: func(order *Order) error {
				order.SerializedOrderCancel = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil confirmation
			setup: func(order *Order) error {
				order.SerializedOrderConfirmation = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil fulfillment
			setup: func(order *Order) error {
				order.SerializedOrderFulfillment = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil complete
			setup: func(order *Order) error {
				order.SerializedOrderComplete = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil dispute open
			setup: func(order *Order) error {
				order.SerializedDisputeOpen = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil dispute close
			setup: func(order *Order) error {
				order.SerializedDisputeClosed = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil dispute update
			setup: func(order *Order) error {
				order.SerializedDisputeUpdate = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil refund
			setup: func(order *Order) error {
				order.SerializedRefunds = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
		{
			// Non nil payment finalized
			setup: func(order *Order) error {
				order.SerializedPaymentFinalized = []byte{0x00}
				err := order.PutMessage(&pb.OrderOpen{
					BuyerID: &pb.ID{
						PeerID: "QmT5NvUtoM5nWFfrQdVrFtvGfKFmG7AHE8P34isapyhCxX",
					},
				})
				return err
			},
			ourID:      "QmPFZPt6FJMZFQABX44RnxmZGh2XGW8ev7KKEMpL8YMxd4",
			canConfirm: false,
		},
	}

	for i, test := range tests {
		var order Order
		if err := test.setup(&order); err != nil {
			t.Errorf("Test %d setup failed: %s", i, err)
		}

		pid, err := peer.IDB58Decode(test.ourID)
		if err != nil {
			t.Errorf("Test %d peerID decode error: %s", i, err)
		}

		canConfirm := order.CanConfirm(pid)
		if canConfirm != test.canConfirm {
			t.Errorf("Got incorrect result. Expected %t, got %t", test.canConfirm, canConfirm)
		}
	}
}

func TestOrder_IsFunded(t *testing.T) {
	tests := []struct {
		setup    func(order *Order) error
		isFunded bool
	}{
		// Funded
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Amount:  "1000",
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}

				return order.PutTransaction(iwallet.Transaction{
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("1000"),
						},
					},
				})
			},
			isFunded: true,
		},
		// Multiple payments
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Amount:  "1000",
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}
				err = order.PutTransaction(iwallet.Transaction{
					ID: "123",
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("100"),
						},
					},
				})
				if err != nil {
					return err
				}

				return order.PutTransaction(iwallet.Transaction{
					ID: "456",
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("900"),
						},
					},
				})
			},
			isFunded: true,
		},
		// Short
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Amount:  "1000",
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}

				return order.PutTransaction(iwallet.Transaction{
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("100"),
						},
					},
				})
			},
			isFunded: false,
		},
	}

	for i, test := range tests {
		var order Order
		if err := test.setup(&order); err != nil {
			t.Errorf("Test %d setup failed: %s", i, err)
		}

		isFunded, err := order.IsFunded()
		if err != nil {
			t.Errorf("Test %d: Is funded error: %s", i, err)
		}
		if isFunded != test.isFunded {
			t.Errorf("Got incorrect result. Expected %t, got %t", test.isFunded, isFunded)
		}
	}
}

func TestOrder_FundingTotal(t *testing.T) {
	tests := []struct {
		setup func(order *Order) error
		total iwallet.Amount
	}{
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}

				return order.PutTransaction(iwallet.Transaction{
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("1000"),
						},
					},
				})
			},
			total: iwallet.NewAmount(1000),
		},
		// Multiple payments
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Amount:  "1000",
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}
				err = order.PutTransaction(iwallet.Transaction{
					ID: "123",
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("100"),
						},
					},
				})
				if err != nil {
					return err
				}

				return order.PutTransaction(iwallet.Transaction{
					ID: "456",
					To: []iwallet.SpendInfo{
						{
							Address: iwallet.NewAddress("aaaaaa", iwallet.CtTestnetMock),
							Amount:  iwallet.NewAmount("900"),
						},
					},
				})
			},
			total: iwallet.NewAmount(1000),
		},
		// No payments
		{
			setup: func(order *Order) error {
				err := order.PutMessage(&pb.OrderOpen{
					Payment: &pb.OrderOpen_Payment{
						Amount:  "1000",
						Address: "aaaaaa",
					},
				})
				if err != nil {
					return err
				}
				return nil
			},
			total: iwallet.NewAmount(0),
		},
	}

	for i, test := range tests {
		var order Order
		if err := test.setup(&order); err != nil {
			t.Errorf("Test %d setup failed: %s", i, err)
		}

		total, err := order.FundingTotal()
		if err != nil {
			t.Errorf("Test %d: Is funded error: %s", i, err)
		}
		if total.Cmp(test.total) != 0 {
			t.Errorf("Got incorrect result. Expected %s, got %s", test.total, total)
		}
	}
}

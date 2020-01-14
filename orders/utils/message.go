package utils

import (
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// MustWrapOrderMessage is a test helper to wrap an order message.
func MustWrapOrderMessage(message proto.Message) *npb.OrderMessage {
	a, err := ptypes.MarshalAny(message)
	if err != nil {
		panic(err)
	}
	var messageType npb.OrderMessage_MessageType
	switch message.(type) {
	case *pb.OrderOpen:
		messageType = npb.OrderMessage_ORDER_OPEN
	case *pb.OrderReject:
		messageType = npb.OrderMessage_ORDER_REJECT
	case *pb.OrderCancel:
		messageType = npb.OrderMessage_ORDER_CANCEL
	case *pb.OrderConfirmation:
		messageType = npb.OrderMessage_ORDER_CONFIRMATION
	case *pb.OrderFulfillment:
		messageType = npb.OrderMessage_ORDER_FULFILLMENT
	case *pb.OrderComplete:
		messageType = npb.OrderMessage_ORDER_COMPLETE
	case *pb.DisputeOpen:
		messageType = npb.OrderMessage_DISPUTE_OPEN
	case *pb.DisputeUpdate:
		messageType = npb.OrderMessage_DISPUTE_UPDATE
	case *pb.DisputeClose:
		messageType = npb.OrderMessage_DISPUTE_CLOSE
	case *pb.Refund:
		messageType = npb.OrderMessage_REFUND
	case *pb.PaymentSent:
		messageType = npb.OrderMessage_PAYMENT_SENT
	case *pb.PaymentFinalized:
		messageType = npb.OrderMessage_PAYMENT_FINALIZED
	}
	return &npb.OrderMessage{
		OrderID:     "abc",
		Message:     a,
		MessageType: messageType,
		Signature:   []byte("1234"),
	}
}

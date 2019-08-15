package models

import (
	"errors"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/proto"
)

var ErrMessageDoesNotExist = errors.New("order message not saved in order")

// IsMessageNotExistError returns whether or not the provided error is a
// ErrMessageDoesNotExist error.
func IsMessageNotExistError(err error) bool {
	return err == ErrMessageDoesNotExist
}

// OrderID is an OpenBazaar order ID.
type OrderID string

// String returns the string representation of the ID.
func (id OrderID) String() string {
	return string(id)
}

// Order holds the state of all orders. This model is saved in the
// database indexed by the order ID.
type Order struct {
	ID OrderID `gorm:"primary_key"`

	SerializedOrderOpen []byte
	OrderOpenAcked      bool

	SerializedOrderReject []byte
	OrderRejectAcked      bool

	SerializedOrderCancel []byte
	OrderCancelAcked      bool

	SerializedOrderConfirmation []byte
	OrderConfirmationAcked      bool

	SerializedOrderFulfillment []byte
	OrderFulfillmentAcked      bool

	SerializedOrderComplete []byte
	OrderCompleteAcked      bool

	SerializedDisputeOpen []byte
	DisputeOpenAcked      bool

	SerializedDisputeUpdate []byte
	DisputeUpdateAcked      bool

	SerializedDisputeClosed []byte
	DisputeClosedAcked      bool

	SerializedRefund []byte
	RefundAcked      bool

	SerializedPaymentSent []byte
	PaymentSentAcked      bool

	SerializedPaymentFinalized []byte
	PaymentFinalizedAcked      bool

	ParkedMessages  []byte
	ErroredMessages []byte
}

// OrderOpenMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderOpenMessage() (*pb.OrderOpen, error) {
	if o.SerializedOrderOpen == nil || len(o.SerializedOrderOpen) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderOpen := new(pb.OrderOpen)
	if err := proto.Unmarshal(o.SerializedOrderOpen, orderOpen); err != nil {
		return nil, err
	}
	return orderOpen, nil
}

// OrderRejectMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderRejectMessage() (*pb.OrderReject, error) {
	if o.SerializedOrderReject == nil || len(o.SerializedOrderReject) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderReject := new(pb.OrderReject)
	if err := proto.Unmarshal(o.SerializedOrderReject, orderReject); err != nil {
		return nil, err
	}
	return orderReject, nil
}

// OrderCancelMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderCancelMessage() (*pb.OrderCancel, error) {
	if o.SerializedOrderCancel == nil || len(o.SerializedOrderCancel) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderCancel := new(pb.OrderCancel)
	if err := proto.Unmarshal(o.SerializedOrderCancel, orderCancel); err != nil {
		return nil, err
	}
	return orderCancel, nil
}

// OrderConfirmationMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderConfirmationMessage() (*pb.OrderConfirmation, error) {
	if o.SerializedOrderConfirmation == nil || len(o.SerializedOrderConfirmation) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderConfirmation := new(pb.OrderConfirmation)
	if err := proto.Unmarshal(o.SerializedOrderConfirmation, orderConfirmation); err != nil {
		return nil, err
	}
	return orderConfirmation, nil
}

// OrderFulfillmentMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderFulfillmentMessage() (*pb.OrderFulfillment, error) {
	if o.SerializedOrderFulfillment == nil || len(o.SerializedOrderFulfillment) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderFulfillment := new(pb.OrderFulfillment)
	if err := proto.Unmarshal(o.SerializedOrderFulfillment, orderFulfillment); err != nil {
		return nil, err
	}
	return orderFulfillment, nil
}

// OrderCompleteMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderCompleteMessage() (*pb.OrderComplete, error) {
	if o.SerializedOrderComplete == nil || len(o.SerializedOrderComplete) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderComplete := new(pb.OrderComplete)
	if err := proto.Unmarshal(o.SerializedOrderComplete, orderComplete); err != nil {
		return nil, err
	}
	return orderComplete, nil
}

// DisputeOpenMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) DisputeOpenMessage() (*pb.DisputeOpen, error) {
	if o.SerializedDisputeOpen == nil || len(o.SerializedDisputeOpen) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	disputeOpen := new(pb.DisputeOpen)
	if err := proto.Unmarshal(o.SerializedDisputeOpen, disputeOpen); err != nil {
		return nil, err
	}
	return disputeOpen, nil
}

// DisputeUpdateMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) DisputeUpdateMessage() (*pb.DisputeUpdate, error) {
	if o.SerializedDisputeUpdate == nil || len(o.SerializedDisputeUpdate) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	disputeUpdate := new(pb.DisputeUpdate)
	if err := proto.Unmarshal(o.SerializedDisputeUpdate, disputeUpdate); err != nil {
		return nil, err
	}
	return disputeUpdate, nil
}

// DisputeClosedMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) DisputeClosedMessage() (*pb.DisputeClose, error) {
	if o.SerializedDisputeClosed == nil || len(o.SerializedDisputeClosed) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	disputeClose := new(pb.DisputeClose)
	if err := proto.Unmarshal(o.SerializedDisputeClosed, disputeClose); err != nil {
		return nil, err
	}
	return disputeClose, nil
}

// RefundMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) RefundMessage() (*pb.Refund, error) {
	if o.SerializedRefund == nil || len(o.SerializedRefund) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	refund := new(pb.Refund)
	if err := proto.Unmarshal(o.SerializedRefund, refund); err != nil {
		return nil, err
	}
	return refund, nil
}

// PaymentSentMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) PaymentSentMessage() (*pb.PaymentSent, error) {
	if o.SerializedPaymentSent == nil || len(o.SerializedPaymentSent) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	paymentSent := new(pb.PaymentSent)
	if err := proto.Unmarshal(o.SerializedPaymentSent, paymentSent); err != nil {
		return nil, err
	}
	return paymentSent, nil
}

// PaymentFinalizedMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) PaymentFinalizedMessage() (*pb.PaymentFinalized, error) {
	if o.SerializedPaymentFinalized == nil || len(o.SerializedPaymentFinalized) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	paymentFinalized := new(pb.PaymentFinalized)
	if err := proto.Unmarshal(o.SerializedPaymentFinalized, paymentFinalized); err != nil {
		return nil, err
	}
	return paymentFinalized, nil
}

// PutMessage serializes the message and saves it in the object at
// the correct location.
func (o *Order) PutMessage(message proto.Message) error {
	ser, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	switch message.(type) {
	case *pb.OrderOpen:
		o.SerializedOrderOpen = ser
	case *pb.OrderReject:
		o.SerializedOrderReject = ser
	case *pb.OrderCancel:
		o.SerializedOrderCancel = ser
	case *pb.OrderConfirmation:
		o.SerializedOrderConfirmation = ser
	case *pb.OrderFulfillment:
		o.SerializedOrderFulfillment = ser
	case *pb.OrderComplete:
		o.SerializedOrderComplete = ser
	case *pb.DisputeOpen:
		o.SerializedDisputeOpen = ser
	case *pb.DisputeUpdate:
		o.SerializedDisputeUpdate = ser
	case *pb.DisputeClose:
		o.SerializedDisputeClosed = ser
	case *pb.Refund:
		o.SerializedRefund = ser
	case *pb.PaymentSent:
		o.SerializedPaymentSent = ser
	case *pb.PaymentFinalized:
		o.SerializedPaymentFinalized = ser
	}
	return nil
}

// ParkMessage adds the message to our list of parked messages.
func (o *Order) ParkMessage(message *npb.OrderMessage) error {
	parkedMessages := new(npb.OrderList)
	if err := proto.Unmarshal(o.ParkedMessages, parkedMessages); err != nil {
		return err
	}
	parkedMessages.Messages = append(parkedMessages.Messages, message)
	ser, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	o.ParkedMessages = ser
	return nil
}

// GetParkedMessages gets the parked messages associated with this order.
func (o *Order) GetParkedMessages() ([]*npb.OrderMessage, error) {
	parkedMessages := new(npb.OrderList)
	if err := proto.Unmarshal(o.ParkedMessages, parkedMessages); err != nil {
		return nil, err
	}
	return parkedMessages.Messages, nil
}

// PutErrorMessage adds the message to our list of errored messages.
func (o *Order) PutErrorMessage(message *npb.OrderMessage) error {
	erroredMessages := new(npb.OrderList)
	if err := proto.Unmarshal(o.ErroredMessages, erroredMessages); err != nil {
		return err
	}
	erroredMessages.Messages = append(erroredMessages.Messages, message)
	ser, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	o.ErroredMessages = ser
	return nil
}

// GetErroredMessages gets the errored messages associated with this order.
func (o *Order) GetErroredMessages() ([]*npb.OrderMessage, error) {
	erroredMessages := new(npb.OrderList)
	if err := proto.Unmarshal(o.ErroredMessages, erroredMessages); err != nil {
		return nil, err
	}
	return erroredMessages.Messages, nil
}

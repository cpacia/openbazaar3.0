package models

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/peer"
	"time"
)

var (
	// ErrMessageDoesNotExist signifies the order message does not exist in the order.
	ErrMessageDoesNotExist = errors.New("message not saved in order")

	// ErrDuplicateTransaction signifies a duplicate transaction was saved in the order.
	ErrDuplicateTransaction = errors.New("duplicate transaction")

	marshaler = jsonpb.Marshaler{
		EmitDefaults: true,
		Indent:       "    ",
	}
)

// IsMessageNotExistError returns whether or not the provided error is a
// ErrMessageDoesNotExist error.
func IsMessageNotExistError(err error) bool {
	return err == ErrMessageDoesNotExist
}

// IsDuplicateTransactionError returns whether or not the provided error is a
// ErrDuplicateTransaction error.
func IsDuplicateTransactionError(err error) bool {
	return err == ErrDuplicateTransaction
}

// OrderID is an OpenBazaar order ID.
type OrderID string

// String returns the string representation of the ID.
func (id OrderID) String() string {
	return string(id)
}

// OrderRole specifies this node's role in the order.
type OrderRole string

const (
	// RoleUnknown means we haven't yet determined the role.
	RoleUnknown OrderRole = "unknown"
	// RoleBuyer represents a buyer.
	RoleBuyer OrderRole = "buyer"
	// RoleVendor represents a vendor.
	RoleVendor OrderRole = "vendor"
	// RoleModerator represents a moderator.
	RoleModerator OrderRole = "moderator"
)

// Order holds the state of all orders. This model is saved in the
// database indexed by the order ID.
type Order struct {
	ID OrderID `gorm:"primary_key"`

	PaymentAddress string `gorm:"index"`

	Transactions []byte

	MyRole string

	Open bool `gorm:"index"`

	LastCheckForPayments time.Time
	RescanPerformed      bool

	SerializedOrderOpen []byte
	OrderOpenSignature  string
	OrderOpenAcked      bool

	SerializedOrderReject []byte
	OrderRejectSignature  string
	OrderRejectAcked      bool

	SerializedOrderCancel []byte
	OrderCancelSignature  string
	OrderCancelAcked      bool

	SerializedOrderConfirmation []byte
	OrderConfirmationSignature  string
	OrderConfirmationAcked      bool

	SerializedRatingSignatures []byte
	RatingSignaturesSignature  string
	RatingSignaturesAcked      bool

	SerializedOrderComplete []byte
	OrderCompleteSignature  string
	OrderCompleteAcked      bool

	SerializedDisputeOpen      []byte
	DisputeOpenSignature       string
	DisputeOpenOtherPartyAcked bool
	DisputeOpenModeratorAcked  bool

	SerializedDisputeUpdate []byte
	DisputeUpdateSignature  string
	DisputeUpdateAcked      bool

	SerializedDisputeClosed []byte
	DisputeClosedSignature  string
	DisputeClosedAcked      bool

	SerializedPaymentFinalized []byte
	PaymentFinalizedSignature  string
	PaymentFinalizedAcked      bool

	SerializedOrderFulfillments []byte
	OrderFulfillmentAcked       bool

	SerializedRefunds []byte
	RefundAcked       bool

	SerializedPaymentSent []byte
	PaymentSentAcked      bool

	ParkedMessages  []byte
	ErroredMessages []byte
}

// Role returns the role of the user for this order.
func (o *Order) Role() OrderRole {
	return OrderRole(o.MyRole)
}

// SetRole sets the role of the user for this order.
func (o *Order) SetRole(role OrderRole) {
	o.MyRole = string(role)
}

// Buyer returns the peer ID of the buyer for this order.
func (o *Order) Buyer() (peer.ID, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return "", err
	}
	return peer.Decode(orderOpen.BuyerID.PeerID)
}

// Vendor returns the peer ID of the vendor for this order.
func (o *Order) Vendor() (peer.ID, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return "", err
	}
	return peer.Decode(orderOpen.Listings[0].Listing.VendorID.PeerID)
}

// Moderator returns the peer ID of the moderator for this order.
func (o *Order) Moderator() (peer.ID, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return "", err
	}
	if orderOpen.Payment.Moderator == "" {
		return "", errors.New("no moderator for order")
	}
	return peer.Decode(orderOpen.Payment.Moderator)
}

// Timestamp returns the timestamp at which this order was opened.
func (o *Order) Timestamp() (time.Time, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return time.Time{}, err
	}
	return ptypes.Timestamp(orderOpen.Timestamp)
}

// GetTransactions returns all the transactions associated with this order.
func (o *Order) GetTransactions() ([]iwallet.Transaction, error) {
	if o.Transactions == nil || len(o.Transactions) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	var transactions []iwallet.Transaction
	if err := json.Unmarshal(o.Transactions, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

// PutTransaction appends the transaction to the order.
func (o *Order) PutTransaction(transaction iwallet.Transaction) error {
	var transactions []iwallet.Transaction
	if o.Transactions != nil {
		if err := json.Unmarshal(o.Transactions, &transactions); err != nil {
			return err
		}
	}

	// Check if the transaction already exists.
	for _, tx := range transactions {
		if tx.ID == transaction.ID {
			return ErrDuplicateTransaction
		}
	}

	transactions = append(transactions, transaction)

	ser, err := json.MarshalIndent(transactions, "", "    ")
	if err != nil {
		return err
	}
	o.Transactions = ser
	return nil
}

// OrderOpenMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderOpenMessage() (*pb.OrderOpen, error) {
	if o.SerializedOrderOpen == nil || len(o.SerializedOrderOpen) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderOpen := new(pb.OrderOpen)
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderOpen), orderOpen); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderReject), orderReject); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderCancel), orderCancel); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderConfirmation), orderConfirmation); err != nil {
		return nil, err
	}
	return orderConfirmation, nil
}

// RatingSignaturesMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) RatingSignaturesMessage() (*pb.RatingSignatures, error) {
	if o.SerializedRatingSignatures == nil || len(o.SerializedRatingSignatures) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	ratingSignatures := new(pb.RatingSignatures)
	if err := jsonpb.UnmarshalString(string(o.SerializedRatingSignatures), ratingSignatures); err != nil {
		return nil, err
	}
	return ratingSignatures, nil
}

// OrderFulfillmentMessage returns the unmarshalled proto objects if they exists in the order.
func (o *Order) OrderFulfillmentMessages() ([]*pb.OrderFulfillment, error) {
	if o.SerializedOrderFulfillments == nil || len(o.SerializedOrderFulfillments) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	fulfillmentList := new(pb.FulfillmentList)
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderFulfillments), fulfillmentList); err != nil {
		return nil, err
	}
	fulfillments := make([]*pb.OrderFulfillment, 0, len(fulfillmentList.Messages))
	for _, m := range fulfillmentList.Messages {
		fulfillments = append(fulfillments, m.FulfillmentMessage)
	}
	return fulfillments, nil
}

// OrderCompleteMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderCompleteMessage() (*pb.OrderComplete, error) {
	if o.SerializedOrderComplete == nil || len(o.SerializedOrderComplete) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderComplete := new(pb.OrderComplete)
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderComplete), orderComplete); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedDisputeOpen), disputeOpen); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedDisputeUpdate), disputeUpdate); err != nil {
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
	if err := jsonpb.UnmarshalString(string(o.SerializedDisputeClosed), disputeClose); err != nil {
		return nil, err
	}
	return disputeClose, nil
}

// RefundMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) Refunds() ([]*pb.Refund, error) {
	if o.SerializedRefunds == nil || len(o.SerializedRefunds) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	refundList := new(pb.RefundList)
	if err := jsonpb.UnmarshalString(string(o.SerializedRefunds), refundList); err != nil {
		return nil, err
	}
	refunds := make([]*pb.Refund, 0, len(refundList.Messages))
	for _, m := range refundList.Messages {
		refunds = append(refunds, m.RefundMessage)
	}
	return refunds, nil
}

// PaymentSentMessages returns a list of PaymentSent objects.
func (o *Order) PaymentSentMessages() ([]*pb.PaymentSent, error) {
	if o.SerializedPaymentSent == nil || len(o.SerializedPaymentSent) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	paymentList := new(pb.PaymentSentList)
	if err := jsonpb.UnmarshalString(string(o.SerializedPaymentSent), paymentList); err != nil {
		return nil, err
	}
	payments := make([]*pb.PaymentSent, 0, len(paymentList.Messages))
	for _, m := range paymentList.Messages {
		payments = append(payments, m.PaymentSentMessage)
	}
	return payments, nil
}

// PaymentFinalizedMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) PaymentFinalizedMessage() (*pb.PaymentFinalized, error) {
	if o.SerializedPaymentFinalized == nil || len(o.SerializedPaymentFinalized) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	paymentFinalized := new(pb.PaymentFinalized)
	if err := jsonpb.UnmarshalString(string(o.SerializedPaymentFinalized), paymentFinalized); err != nil {
		return nil, err
	}
	return paymentFinalized, nil
}

// PutMessage serializes the message and saves it in the object at
// the correct location.
func (o *Order) PutMessage(message *npb.OrderMessage) error {
	sig := base64.StdEncoding.EncodeToString(message.Signature)
	var (
		msg        proto.Message
		setMessage func(ser []byte)
	)

	switch message.MessageType {
	case npb.OrderMessage_ORDER_OPEN:
		msg = new(pb.OrderOpen)
		setMessage = func(ser []byte) { o.SerializedOrderOpen = ser }
		o.OrderOpenSignature = sig
	case npb.OrderMessage_ORDER_REJECT:
		msg = new(pb.OrderReject)
		setMessage = func(ser []byte) { o.SerializedOrderReject = ser }
		o.OrderRejectSignature = sig
	case npb.OrderMessage_ORDER_CANCEL:
		msg = new(pb.OrderCancel)
		setMessage = func(ser []byte) { o.SerializedOrderCancel = ser }
		o.OrderCancelSignature = sig
	case npb.OrderMessage_ORDER_CONFIRMATION:
		msg = new(pb.OrderConfirmation)
		setMessage = func(ser []byte) { o.SerializedOrderConfirmation = ser }
		o.OrderConfirmationSignature = sig
	case npb.OrderMessage_RATING_SIGNATURES:
		msg = new(pb.RatingSignatures)
		setMessage = func(ser []byte) { o.SerializedRatingSignatures = ser }
		o.RatingSignaturesSignature = sig
	case npb.OrderMessage_ORDER_COMPLETE:
		msg = new(pb.OrderComplete)
		setMessage = func(ser []byte) { o.SerializedOrderComplete = ser }
		o.OrderCompleteSignature = sig
	case npb.OrderMessage_DISPUTE_OPEN:
		msg = new(pb.DisputeOpen)
		setMessage = func(ser []byte) { o.SerializedDisputeOpen = ser }
		o.DisputeOpenSignature = sig
	case npb.OrderMessage_DISPUTE_UPDATE:
		msg = new(pb.DisputeUpdate)
		setMessage = func(ser []byte) { o.SerializedDisputeUpdate = ser }
		o.DisputeUpdateSignature = sig
	case npb.OrderMessage_DISPUTE_CLOSE:
		msg = new(pb.DisputeClose)
		setMessage = func(ser []byte) { o.SerializedDisputeClosed = ser }
		o.DisputeClosedSignature = sig
	case npb.OrderMessage_ORDER_FULFILLMENT:
		fulfillmentMsg := new(pb.OrderFulfillment)
		if err := ptypes.UnmarshalAny(message.Message, fulfillmentMsg); err != nil {
			return err
		}

		fulfillmentList := new(pb.FulfillmentList)
		if o.SerializedOrderFulfillments != nil {
			if err := jsonpb.UnmarshalString(string(o.SerializedOrderFulfillments), fulfillmentList); err != nil {
				return err
			}
		}
		for _, f := range fulfillmentList.Messages {
			for _, item := range f.FulfillmentMessage.Fulfillments {
				for _, fulfilledItems := range fulfillmentMsg.Fulfillments {
					if item.ItemIndex == fulfilledItems.ItemIndex {
						return ErrDuplicateTransaction
					}
				}
			}
		}
		fulfillmentList.Messages = append(fulfillmentList.Messages, &pb.FulfillmentList_Message{
			FulfillmentMessage: fulfillmentMsg,
			Signature:          message.Signature,
		})
		ser, err := marshaler.MarshalToString(fulfillmentList)
		if err != nil {
			return err
		}

		o.SerializedOrderFulfillments = []byte(ser)
		return nil
	case npb.OrderMessage_REFUND:
		refundMsg := new(pb.Refund)
		if err := ptypes.UnmarshalAny(message.Message, refundMsg); err != nil {
			return err
		}

		refundList := new(pb.RefundList)
		if o.SerializedRefunds != nil {
			if err := jsonpb.UnmarshalString(string(o.SerializedRefunds), refundList); err != nil {
				return err
			}
		}
		for _, r := range refundList.Messages {
			if r.RefundMessage.GetTransactionID() != "" && r.RefundMessage.GetTransactionID() == refundMsg.GetTransactionID() {
				return ErrDuplicateTransaction
			}
			if r.RefundMessage.GetReleaseInfo() != nil && refundMsg.GetReleaseInfo() != nil {
				out1, err := marshaler.MarshalToString(r.RefundMessage.GetReleaseInfo())
				if err != nil {
					return err
				}
				out2, err := marshaler.MarshalToString(refundMsg.GetReleaseInfo())
				if err != nil {
					return err
				}

				if out1 == out2 {
					return ErrDuplicateTransaction
				}
			}
		}
		refundList.Messages = append(refundList.Messages, &pb.RefundList_Message{
			RefundMessage: refundMsg,
			Signature:     message.Signature,
		})
		ser, err := marshaler.MarshalToString(refundList)
		if err != nil {
			return err
		}

		o.SerializedRefunds = []byte(ser)
		return nil
	case npb.OrderMessage_PAYMENT_SENT:
		pymtSentMsg := new(pb.PaymentSent)
		if err := ptypes.UnmarshalAny(message.Message, pymtSentMsg); err != nil {
			return err
		}

		paymentList := new(pb.PaymentSentList)
		if o.SerializedPaymentSent != nil {
			if err := jsonpb.UnmarshalString(string(o.SerializedPaymentSent), paymentList); err != nil {
				return err
			}
		}
		for _, m := range paymentList.Messages {
			if m.PaymentSentMessage.TransactionID == pymtSentMsg.TransactionID {
				return ErrDuplicateTransaction
			}
		}
		paymentList.Messages = append(paymentList.Messages, &pb.PaymentSentList_Message{
			PaymentSentMessage: pymtSentMsg,
			Signature:          message.Signature,
		})
		ser, err := marshaler.MarshalToString(paymentList)
		if err != nil {
			return err
		}

		o.SerializedPaymentSent = []byte(ser)
		return nil
	case npb.OrderMessage_PAYMENT_FINALIZED:
		msg = new(pb.PaymentFinalized)
		setMessage = func(ser []byte) { o.SerializedPaymentFinalized = ser }
		o.PaymentFinalizedSignature = sig
	}

	if err := ptypes.UnmarshalAny(message.Message, msg); err != nil {
		return err
	}
	out, err := marshaler.MarshalToString(msg)
	if err != nil {
		return err
	}
	setMessage([]byte(out))
	return nil
}

// ParkMessage adds the message to our list of parked messages.
func (o *Order) ParkMessage(message *npb.OrderMessage) error {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages != nil {
		if err := proto.Unmarshal(o.ParkedMessages, parkedMessages); err != nil {
			return err
		}
	}
	parkedMessages.Messages = append(parkedMessages.Messages, message)
	ser, err := proto.Marshal(parkedMessages)
	if err != nil {
		return err
	}
	o.ParkedMessages = ser
	return nil
}

// DeleteParkedMessage deletes a parked message from the order.
func (o *Order) DeleteParkedMessage(messageType npb.OrderMessage_MessageType) error {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages != nil {
		if err := proto.Unmarshal(o.ParkedMessages, parkedMessages); err != nil {
			return err
		}
	}
	for i, message := range parkedMessages.Messages {
		if message.MessageType == messageType {
			parkedMessages.Messages = append(parkedMessages.Messages[:i], parkedMessages.Messages[i+1:]...)
			break
		}
	}
	ser, err := proto.Marshal(parkedMessages)
	if err != nil {
		return err
	}
	o.ParkedMessages = ser
	return nil
}

// GetParkedMessages gets the parked messages associated with this order.
func (o *Order) GetParkedMessages() ([]*npb.OrderMessage, error) {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages == nil || len(o.ParkedMessages) == 0 {
		return nil, nil
	}
	if err := proto.Unmarshal(o.ParkedMessages, parkedMessages); err != nil {
		return nil, err
	}
	return parkedMessages.Messages, nil
}

// PutErrorMessage adds the message to our list of errored messages.
func (o *Order) PutErrorMessage(message *npb.OrderMessage) error {
	erroredMessages := new(npb.OrderList)
	if o.ErroredMessages != nil {
		if err := proto.Unmarshal(o.ErroredMessages, erroredMessages); err != nil {
			return err
		}
	}
	erroredMessages.Messages = append(erroredMessages.Messages, message)
	ser, err := proto.Marshal(erroredMessages)
	if err != nil {
		return err
	}
	o.ErroredMessages = ser
	return nil
}

// GetErroredMessages gets the errored messages associated with this order.
func (o *Order) GetErroredMessages() ([]*npb.OrderMessage, error) {
	erroredMessages := new(npb.OrderList)
	if o.ErroredMessages == nil || len(o.ErroredMessages) == 0 {
		return nil, nil
	}
	if err := proto.Unmarshal(o.ErroredMessages, erroredMessages); err != nil {
		return nil, err
	}
	return erroredMessages.Messages, nil
}

// CanReject returns whether or not this order is in a state where the user can
// reject the order.
func (o *Order) CanReject() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only vendors can reject.
	if o.Role() != RoleVendor {
		return false
	}

	// Cannot cancel if the order has progressed passed order open.
	if o.SerializedOrderReject != nil || o.SerializedOrderCancel != nil ||
		o.SerializedOrderConfirmation != nil || o.SerializedOrderFulfillments != nil ||
		o.SerializedOrderComplete != nil || o.SerializedDisputeOpen != nil ||
		o.SerializedDisputeUpdate != nil || o.SerializedDisputeClosed != nil ||
		o.SerializedRefunds != nil || o.SerializedPaymentFinalized != nil {

		return false
	}
	return true
}

// CanConfirm returns whether or not this order is in a state where the user can
// confirm the order.
func (o *Order) CanConfirm() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only vendors can confirm.
	if o.Role() != RoleVendor {
		return false
	}

	// Cannot confirm if the order has progressed passed order open.
	if o.SerializedOrderReject != nil || o.SerializedOrderCancel != nil ||
		o.SerializedOrderConfirmation != nil || o.SerializedOrderFulfillments != nil ||
		o.SerializedOrderComplete != nil || o.SerializedDisputeOpen != nil ||
		o.SerializedDisputeUpdate != nil || o.SerializedDisputeClosed != nil ||
		o.SerializedRefunds != nil || o.SerializedPaymentFinalized != nil {

		return false
	}
	return true
}

// CanCancel returns whether or not this order is in a state where the user can
// cancel the order.
func (o *Order) CanCancel() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only buyers can confirm.
	if o.Role() != RoleBuyer {
		return false
	}

	// Cannot cancel if the order has progressed passed order open.
	if o.SerializedOrderReject != nil || o.SerializedOrderCancel != nil ||
		o.SerializedOrderConfirmation != nil || o.SerializedOrderFulfillments != nil ||
		o.SerializedOrderComplete != nil || o.SerializedDisputeOpen != nil ||
		o.SerializedDisputeUpdate != nil || o.SerializedDisputeClosed != nil ||
		o.SerializedRefunds != nil || o.SerializedPaymentFinalized != nil {

		return false
	}
	return true
}

// CanRefund returns whether or not this order is in a state where the user can
// refund the order.
func (o *Order) CanRefund() bool {
	// OrderOpen must exist.
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only vendors can refund.
	if o.Role() != RoleVendor {
		return false
	}

	// Can't refund cancelable.
	if orderOpen.Payment == nil || orderOpen.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
		return false
	}

	// Cannot refund if the order has been completed or canceled.
	if o.SerializedOrderComplete != nil || o.SerializedPaymentFinalized != nil || o.SerializedOrderCancel != nil {
		return false
	}

	return true
}

// CanFulfill returns whether or not this order is in a state where the user can
// fulfill the order.
func (o *Order) CanFulfill() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only vendors can fulfill.
	if o.Role() != RoleVendor {
		return false
	}

	// Order must have been confirmed.
	if o.SerializedOrderConfirmation == nil {
		return false
	}

	// Order must be funded.
	funded, err := o.IsFunded()
	if err != nil {
		return false
	}

	if !funded {
		return false
	}

	// Order must not be fulfilled already.
	fulfilled, err := o.IsFulfilled()
	if err != nil {
		return false
	}

	if fulfilled {
		return false
	}

	// Cannot fulfill if the order has been completed or canceled.
	if o.SerializedOrderComplete != nil || o.SerializedPaymentFinalized != nil || o.SerializedOrderCancel != nil {
		return false
	}

	return true
}

// CanComplete returns whether or not this order is in a state where the user can
// complete the order and leave a rating.
func (o *Order) CanComplete() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only buyers can complete.
	if o.Role() != RoleBuyer {
		return false
	}

	fulfilled, err := o.IsFulfilled()
	if err != nil {
		return false
	}

	// Order must be fulfilled
	if !fulfilled {
		return false
	}

	// Cannot complete if the order has been completed.
	if o.SerializedOrderComplete != nil || o.SerializedPaymentFinalized != nil {
		return false
	}

	// Cannot complete if a dispute is open.
	if o.UnderActiveDispute() {
		return false
	}

	return true
}

// CanDispute returns whether or not this order is in a state where the user can
// dispute the order.
func (o *Order) CanDispute() bool {
	// OrderOpen must exist.
	_, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}

	// Only buyers and vendors can dispute.
	if o.Role() != RoleBuyer && o.Role() != RoleVendor {
		return false
	}

	if o.Role() == RoleVendor {
		fulfilled, err := o.IsFulfilled()
		if err != nil {
			return false
		}

		// Vendor must fulfill order prior to disputing.
		if !fulfilled {
			return false
		}
	}

	// Cannot dispute if the order has been completed.
	if o.SerializedOrderComplete != nil || o.SerializedPaymentFinalized != nil {
		return false
	}

	// Cannot dispute if a dispute is open.
	if o.UnderActiveDispute() {
		return false
	}

	return true
}

// UnderActiveDispute returns whether this order is currently being disputed.
func (o *Order) UnderActiveDispute() bool {
	if o.SerializedDisputeOpen != nil && o.SerializedDisputeClosed == nil {
		return true
	}
	return false
}

// IsFunded returns whether this order is fully funded or not.
func (o *Order) IsFunded() (bool, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return false, err
	}

	var (
		requestedAmount = iwallet.NewAmount(orderOpen.Payment.Amount)
		paymentAddress  = orderOpen.Payment.Address
		totalPaid       iwallet.Amount
	)

	txs, err := o.GetTransactions()
	if err != nil && !IsMessageNotExistError(err) {
		return false, err
	}
	for _, tx := range txs {
		for _, to := range tx.To {
			if to.Address.String() == paymentAddress {
				totalPaid = totalPaid.Add(to.Amount)
			}
		}
	}
	return totalPaid.Cmp(requestedAmount) >= 0, nil
}

// IsFulfilled returns whether a fulfillment message is saved for each item in the order.
func (o *Order) IsFulfilled() (bool, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return false, err
	}

	m := make(map[int]bool)

	for i := range orderOpen.Items {
		m[i] = true
	}

	fulfillments, err := o.OrderFulfillmentMessages()
	if err != nil && !IsMessageNotExistError(err) {
		return false, err
	}

	for _, f := range fulfillments {
		for _, f2 := range f.Fulfillments {
			delete(m, int(f2.ItemIndex))
		}
	}

	return len(m) == 0, nil
}

// FundingTotal returns the total amount paid to this order.
func (o *Order) FundingTotal() (iwallet.Amount, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return iwallet.NewAmount(0), err
	}

	var (
		paymentAddress = orderOpen.Payment.Address
		totalPaid      iwallet.Amount
	)

	txs, err := o.GetTransactions()
	if err != nil && !IsMessageNotExistError(err) {
		return iwallet.NewAmount(0), err
	}
	for _, tx := range txs {
		for _, to := range tx.To {
			if to.Address.String() == paymentAddress {
				totalPaid = totalPaid.Add(to.Amount)
			}
		}
	}
	return totalPaid, nil
}

// MarshalBinary returns a serialized protobuf format.
func (o *Order) MarshalBinary() ([]byte, error) {
	contract, err := o.toProtobuf()
	if err != nil {
		return nil, err
	}

	return proto.Marshal(contract)
}

// MarshalJSON provides custom JSON marshalling for the order model. Since this method is primarily
// used to return data to the API, this is the appropriate place to normalize the data to the format
// the API is expecting.
func (o *Order) MarshalJSON() ([]byte, error) {
	contract, err := o.toProtobuf()
	if err != nil {
		return nil, err
	}

	out, err := marshaler.MarshalToString(contract)
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

func (o *Order) toProtobuf() (*pb.Contract, error) {
	contract := pb.Contract{
		OrderID: o.ID.String(),
		Role:    string(o.Role()),
	}

	var err error
	contract.OrderOpen, err = o.OrderOpenMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.OrderReject, err = o.OrderRejectMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.OrderCancel, err = o.OrderCancelMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.OrderConfirmation, err = o.OrderConfirmationMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.OrderComplete, err = o.OrderCompleteMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.DisputeOpen, err = o.DisputeOpenMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.DisputeClose, err = o.DisputeClosedMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.DisputeUpdate, err = o.DisputeUpdateMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.PaymentFinalized, err = o.PaymentFinalizedMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.OrderFulfillments, err = o.OrderFulfillmentMessages()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.Refunds, err = o.Refunds()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	contract.PaymentsSent, err = o.PaymentSentMessages()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}

	if o.ParkedMessages != nil {
		parked := new(npb.OrderList)
		if err := json.Unmarshal(o.ParkedMessages, parked); err != nil {
			return nil, err
		}
		contract.ParkedMessages = parked
	}

	if o.ErroredMessages != nil {
		errored := new(npb.OrderList)
		if err := json.Unmarshal(o.ParkedMessages, errored); err != nil {
			return nil, err
		}
		contract.ErroredMessages = errored
	}

	var transactions []*pb.Contract_Transaction
	txs, err := o.GetTransactions()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return nil, err
	}
	for _, tx := range txs {
		ts, err := ptypes.TimestampProto(tx.Timestamp)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &pb.Contract_Transaction{
			Txid:      tx.ID.String(),
			Value:     tx.Value.String(),
			Timestamp: ts,
		})
	}
	contract.Transactions = transactions

	contract.OrderOpenAcked = o.OrderOpenAcked
	contract.OrderRejectAcked = o.OrderRejectAcked
	contract.OrderCancelAcked = o.OrderCancelAcked
	contract.OrderConfirmationAcked = o.OrderConfirmationAcked
	contract.OrderCompleteAcked = o.OrderCompleteAcked
	contract.DisputeUpdateAcked = o.DisputeUpdateAcked
	contract.DisputeCloseAcked = o.DisputeClosedAcked
	contract.PaymentFinalizedAcked = o.PaymentFinalizedAcked
	contract.FulfillmentsAcked = o.OrderFulfillmentAcked
	contract.RefundsAcked = o.RefundAcked
	contract.PaymentsSentAcked = o.PaymentSentAcked

	if contract.DisputeOpen != nil && (contract.DisputeOpen.OpenedBy == pb.DisputeOpen_BUYER && o.Role() == RoleBuyer ||
		contract.DisputeOpen.OpenedBy == pb.DisputeOpen_VENDOR && o.Role() == RoleVendor) {
		contract.DisputeOpenOtherPartyAcked = o.DisputeOpenOtherPartyAcked
		contract.DisputeOpenModeratorAcked = o.DisputeOpenModeratorAcked
	}
	return &contract, nil
}

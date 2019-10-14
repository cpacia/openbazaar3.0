package models

import (
	"encoding/json"
	"errors"
	"github.com/OpenBazaar/jsonpb"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
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
type OrderRole uint8

const (
	// RoleUnknown means we haven't yet determined the role.
	RoleUnknown OrderRole = iota
	// RoleBuyer represents a buyer.
	RoleBuyer
	// RoleVendor represents a vendor.
	RoleVendor
	// RoleModerator represents a moderator.
	RoleModerator
)

// Order holds the state of all orders. This model is saved in the
// database indexed by the order ID.
type Order struct {
	ID OrderID `gorm:"primary_key"`

	PaymentAddress string `gorm:"index"`

	Transactions []byte

	MyRole uint8

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

// Role returns the role of the user for this order.
func (o *Order) Role() OrderRole {
	return OrderRole(o.MyRole)
}

// SetRole sets the role of the user for this order.
func (o *Order) SetRole(role OrderRole) {
	o.MyRole = uint8(role)
}

// Buyer returns the peer ID of the buyer for this order.
func (o *Order) Buyer() (peer.ID, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return "", err
	}
	return peer.IDB58Decode(orderOpen.BuyerID.PeerID)
}

// Vendor returns the peer ID of the vendor for this order.
func (o *Order) Vendor() (peer.ID, error) {
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return "", err
	}
	return peer.IDB58Decode(orderOpen.Listings[0].Listing.VendorID.PeerID)
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
	return peer.IDB58Decode(orderOpen.Payment.Moderator)
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

// OrderFulfillmentMessage returns the unmarshalled proto object if it exists in the order.
func (o *Order) OrderFulfillmentMessage() (*pb.OrderFulfillment, error) {
	if o.SerializedOrderFulfillment == nil || len(o.SerializedOrderFulfillment) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	orderFulfillment := new(pb.OrderFulfillment)
	if err := jsonpb.UnmarshalString(string(o.SerializedOrderFulfillment), orderFulfillment); err != nil {
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
func (o *Order) RefundMessage() (*pb.Refund, error) {
	if o.SerializedRefund == nil || len(o.SerializedRefund) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	refund := new(pb.Refund)
	if err := jsonpb.UnmarshalString(string(o.SerializedRefund), refund); err != nil {
		return nil, err
	}
	return refund, nil
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
	payments = append(payments, paymentList.Messages...)

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
func (o *Order) PutMessage(message proto.Message) error {
	s, err := marshaler.MarshalToString(message)
	if err != nil {
		return err
	}
	ser := []byte(s)
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
		paymentList := new(pb.PaymentSentList)
		if o.SerializedPaymentSent != nil {
			if err := jsonpb.UnmarshalString(string(o.SerializedPaymentSent), paymentList); err != nil {
				return err
			}
		}
		for _, m := range paymentList.Messages {
			if m.TransactionID == message.(*pb.PaymentSent).TransactionID {
				return ErrDuplicateTransaction
			}
		}
		paymentList.Messages = append(paymentList.Messages, message.(*pb.PaymentSent))
		ser, err := marshaler.MarshalToString(paymentList)
		if err != nil {
			return err
		}

		o.SerializedPaymentSent = []byte(ser)
	case *pb.PaymentFinalized:
		o.SerializedPaymentFinalized = ser
	}
	return nil
}

// ParkMessage adds the message to our list of parked messages.
func (o *Order) ParkMessage(message *npb.OrderMessage) error {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages != nil {
		if err := jsonpb.UnmarshalString(string(o.ParkedMessages), parkedMessages); err != nil {
			return err
		}
	}
	parkedMessages.Messages = append(parkedMessages.Messages, message)
	ser, err := marshaler.MarshalToString(parkedMessages)
	if err != nil {
		return err
	}
	o.ParkedMessages = []byte(ser)
	return nil
}

// DeleteParkedMessage deletes a parked message from the order.
func (o *Order) DeleteParkedMessage(messageType npb.OrderMessage_MessageType) error {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages != nil {
		if err := jsonpb.UnmarshalString(string(o.ParkedMessages), parkedMessages); err != nil {
			return err
		}
	}
	for i, message := range parkedMessages.Messages {
		if message.MessageType == messageType {
			parkedMessages.Messages = append(parkedMessages.Messages[:i], parkedMessages.Messages[i+1:]...)
			break
		}
	}
	ser, err := marshaler.MarshalToString(parkedMessages)
	if err != nil {
		return err
	}
	o.ParkedMessages = []byte(ser)
	return nil
}

// GetParkedMessages gets the parked messages associated with this order.
func (o *Order) GetParkedMessages() ([]*npb.OrderMessage, error) {
	parkedMessages := new(npb.OrderList)
	if o.ParkedMessages == nil || len(o.ParkedMessages) == 0 {
		return nil, nil
	}
	if err := jsonpb.UnmarshalString(string(o.ParkedMessages), parkedMessages); err != nil {
		return nil, err
	}
	return parkedMessages.Messages, nil
}

// PutErrorMessage adds the message to our list of errored messages.
func (o *Order) PutErrorMessage(message *npb.OrderMessage) error {
	erroredMessages := new(npb.OrderList)
	if o.ErroredMessages != nil {
		if err := jsonpb.UnmarshalString(string(o.ErroredMessages), erroredMessages); err != nil {
			return err
		}
	}
	erroredMessages.Messages = append(erroredMessages.Messages, message)
	ser, err := marshaler.MarshalToString(erroredMessages)
	if err != nil {
		return err
	}
	o.ErroredMessages = []byte(ser)
	return nil
}

// GetErroredMessages gets the errored messages associated with this order.
func (o *Order) GetErroredMessages() ([]*npb.OrderMessage, error) {
	erroredMessages := new(npb.OrderList)
	if o.ErroredMessages == nil || len(o.ErroredMessages) == 0 {
		return nil, nil
	}
	if err := jsonpb.UnmarshalString(string(o.ErroredMessages), erroredMessages); err != nil {
		return nil, err
	}
	return erroredMessages.Messages, nil
}

// CanReject returns whether or not this order is in a state where the user can
// reject the order.
func (o *Order) CanReject(ourPeerID peer.ID) bool {
	// OrderOpen must exist.
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}
	if orderOpen.BuyerID == nil {
		return false
	}
	// Only vendors can reject.
	if orderOpen.BuyerID.PeerID == ourPeerID.Pretty() {
		return false
	}

	// Cannot cancel if the order has progressed passed order open.
	if o.SerializedOrderReject != nil || o.SerializedOrderCancel != nil ||
		o.SerializedOrderConfirmation != nil || o.SerializedOrderFulfillment != nil ||
		o.SerializedOrderComplete != nil || o.SerializedDisputeOpen != nil ||
		o.SerializedDisputeUpdate != nil || o.SerializedDisputeClosed != nil ||
		o.SerializedRefund != nil || o.SerializedPaymentFinalized != nil {

		return false
	}
	return true
}

// CanConfirm returns whether or not this order is in a state where the user can
// confirmed the order.
func (o *Order) CanConfirm(ourPeerID peer.ID) bool {
	// OrderOpen must exist.
	orderOpen, err := o.OrderOpenMessage()
	if err != nil {
		return false
	}
	if orderOpen.BuyerID == nil {
		return false
	}
	// Only vendors can confirm.
	if orderOpen.BuyerID.PeerID == ourPeerID.Pretty() {
		return false
	}

	// Cannot confirm if the order has progressed passed order open.
	if o.SerializedOrderReject != nil || o.SerializedOrderCancel != nil ||
		o.SerializedOrderConfirmation != nil || o.SerializedOrderFulfillment != nil ||
		o.SerializedOrderComplete != nil || o.SerializedDisputeOpen != nil ||
		o.SerializedDisputeUpdate != nil || o.SerializedDisputeClosed != nil ||
		o.SerializedRefund != nil || o.SerializedPaymentFinalized != nil {

		return false
	}
	return true
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
	for _, tx := range txs {
		for _, to := range tx.To {
			if to.Address.String() == paymentAddress {
				totalPaid = totalPaid.Add(to.Amount)
			}
		}
	}
	return totalPaid.Cmp(requestedAmount) >= 0, nil
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
	for _, tx := range txs {
		for _, to := range tx.To {
			if to.Address.String() == paymentAddress {
				totalPaid = totalPaid.Add(to.Amount)
			}
		}
	}
	return totalPaid, nil
}

package events

import (
	"time"
)

// TypedNotification contains a single method which allows
// us to get the type of the notification. All notifications
// should implement this.
type TypedNotification interface {
	// Type returns the type of the notification.
	Type() string
}

type Thumbnail struct {
	Tiny  string `json:"tiny"`
	Small string `json:"small"`
}

type ListingPrice struct {
	Amount           uint64  `json:"amount"`
	CurrencyCode     string  `json:"currencyCode"`
	PriceModifier    float32 `json:"priceModifier"`
	CoinDivisibility uint32  `json:"coinDivisibility"`
}

type OrderNotification struct {
	BuyerHandle string       `json:"buyerHandle"`
	BuyerID     string       `json:"buyerId"`
	ID          string       `json:"notificationId"`
	ListingType string       `json:"listingType"`
	OrderId     string       `json:"orderId"`
	Price       ListingPrice `json:"price"`
	Slug        string       `json:"slug"`
	Thumbnail   Thumbnail    `json:"thumbnail"`
	Title       string       `json:"title"`
}

func (n *OrderNotification) Type() string { return "OrderNotification" }

type PaymentNotification struct {
	ID           string `json:"notificationId"`
	OrderId      string `json:"orderId"`
	FundingTotal uint64 `json:"fundingTotal"`
	CoinType     string `json:"coinType"`
}

func (n *PaymentNotification) Type() string { return "PaymentNotification" }

type OrderConfirmationNotification struct {
	ID           string    `json:"notificationId"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

func (n *OrderConfirmationNotification) Type() string { return "OrderConfirmationNotification" }

type OrderDeclinedNotification struct {
	ID           string    `json:"notificationId"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

func (n *OrderDeclinedNotification) Type() string { return "OrderDeclinedNotification" }

type OrderCancelNotification struct {
	ID          string    `json:"notificationId"`
	OrderId     string    `json:"orderId"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	BuyerHandle string    `json:"buyerHandle"`
	BuyerID     string    `json:"buyerId"`
}

func (n *OrderCancelNotification) Type() string { return "OrderCancelNotification" }

type RefundNotification struct {
	ID           string    `json:"notificationId"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

func (n *RefundNotification) Type() string { return "RefundNotification" }

type FulfillmentNotification struct {
	ID           string    `json:"notificationId"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

func (n *FulfillmentNotification) Type() string { return "FulfillmentNotification" }

type ProcessingErrorNotification struct {
	ID           string    `json:"notificationId"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

func (n *ProcessingErrorNotification) Type() string { return "ProcessingErrorNotification" }

type CompletionNotification struct {
	ID          string    `json:"notificationId"`
	OrderId     string    `json:"orderId"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	BuyerHandle string    `json:"buyerHandle"`
	BuyerID     string    `json:"buyerId"`
}

func (n *CompletionNotification) Type() string { return "CompletionNotification" }

type DisputeOpenNotification struct {
	ID             string    `json:"notificationId"`
	OrderId        string    `json:"orderId"`
	Thumbnail      Thumbnail `json:"thumbnail"`
	DisputerID     string    `json:"disputerId"`
	DisputerHandle string    `json:"disputerHandle"`
	DisputeeID     string    `json:"disputeeId"`
	DisputeeHandle string    `json:"disputeeHandle"`
	Buyer          string    `json:"buyer"`
}

func (n *DisputeOpenNotification) Type() string { return "DisputeOpenNotification" }

type DisputeUpdateNotification struct {
	ID             string    `json:"notificationId"`
	OrderId        string    `json:"orderId"`
	Thumbnail      Thumbnail `json:"thumbnail"`
	DisputerID     string    `json:"disputerId"`
	DisputerHandle string    `json:"disputerHandle"`
	DisputeeID     string    `json:"disputeeId"`
	DisputeeHandle string    `json:"disputeeHandle"`
	Buyer          string    `json:"buyer"`
}

func (n *DisputeUpdateNotification) Type() string { return "DisputeUpdateNotification" }

type DisputeCloseNotification struct {
	ID               string    `json:"notificationId"`
	OrderId          string    `json:"orderId"`
	Thumbnail        Thumbnail `json:"thumbnail"`
	OtherPartyID     string    `json:"otherPartyId"`
	OtherPartyHandle string    `json:"otherPartyHandle"`
	Buyer            string    `json:"buyer"`
}

func (n *DisputeCloseNotification) Type() string { return "DisputeCloseNotification" }

type DisputeAcceptedNotification struct {
	ID               string    `json:"notificationId"`
	OrderId          string    `json:"orderId"`
	Thumbnail        Thumbnail `json:"thumbnail"`
	OherPartyID      string    `json:"otherPartyId"`
	OtherPartyHandle string    `json:"otherPartyHandle"`
	Buyer            string    `json:"buyer"`
}

func (n *DisputeAcceptedNotification) Type() string { return "DisputeAcceptedNotification" }

type FollowNotification struct {
	ID     string `json:"notificationId"`
	PeerID string `json:"peerID"`
}

func (n *FollowNotification) Type() string { return "FollowNotification" }

type UnfollowNotification struct {
	ID     string `json:"notificationId"`
	PeerID string `json:"peerID"`
}

func (n *UnfollowNotification) Type() string { return "UnfollowNotification" }

type ModeratorAddNotification struct {
	ID     string `json:"notificationId"`
	PeerId string `json:"peerId"`
}

func (n *ModeratorAddNotification) Type() string { return "ModeratorAddNotification" }

type ModeratorRemoveNotification struct {
	ID     string `json:"notificationId"`
	PeerId string `json:"peerId"`
}

func (n *ModeratorRemoveNotification) Type() string { return "ModeratorRemoveNotification" }

type StatusNotification struct {
	Status string `json:"status"`
}

func (n *StatusNotification) Type() string { return "StatusNotification" }

// ChatMessageNotification handles serialization of ChatMessages for notifications
type ChatMessageNotification struct {
	MessageID string    `json:"messageID"`
	PeerID    string    `json:"peerID"`
	Subject   string    `json:"subject"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Outgoing  bool      `json:"outgoing"`
	Message   string    `json:"message"`
}

func (n *ChatMessageNotification) Type() string { return "ChatMessageNotification" }

type ChatReadNotification struct {
	MessageID string `json:"messageID"`
	PeerID    string `json:"peerID"`
	Subject   string `json:"subject"`
}

func (n *ChatReadNotification) Type() string { return "ChatReadNotification" }

type ChatTypingNotification struct {
	MessageID string `json:"messageID"`
	PeerID    string `json:"peerID"`
	Subject   string `json:"subject"`
}

func (n *ChatTypingNotification) Type() string { return "ChatTypingNotification" }

type IncomingTransactionNotification struct {
	Wallet        string    `json:"wallet"`
	Txid          string    `json:"txid"`
	Value         int64     `json:"value"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	Memo          string    `json:"memo"`
	Timestamp     time.Time `json:"timestamp"`
	Confirmations int32     `json:"confirmations"`
	OrderId       string    `json:"orderId"`
	Thumbnail     string    `json:"thumbnail"`
	Height        int32     `json:"height"`
	CanBumpFee    bool      `json:"canBumpFee"`
}

func (n *IncomingTransactionNotification) Type() string { return "IncomingTransactionNotification" }

// VendorDisputeTimeout represents a notification about a sale
// which will soon be unable to dispute. The Type indicates the age of the
// purchase and OrderID references the purchases orderID in the database schema
type VendorDisputeTimeoutNotification struct {
	ID        string    `json:"notificationId"`
	OrderID   string    `json:"purchaseOrderId"`
	ExpiresIn uint      `json:"expiresIn"`
	Thumbnail Thumbnail `json:"thumbnail"`
}

func (n *VendorDisputeTimeoutNotification) Type() string { return "VendorDisputeTimeoutNotification" }

// BuyerDisputeTimeout represents a notification about a purchase
// which will soon be unable to dispute.
type BuyerDisputeTimeoutNotification struct {
	ID        string    `json:"notificationId"`
	OrderID   string    `json:"orderId"`
	ExpiresIn uint      `json:"expiresIn"`
	Thumbnail Thumbnail `json:"thumbnail"`
}

func (n *BuyerDisputeTimeoutNotification) Type() string { return "BuyerDisputeTimeoutNotification" }

// BuyerDisputeExpiry represents a notification about a purchase
// which has an open dispute that is expiring
type BuyerDisputeExpiryNotification struct {
	ID        string    `json:"notificationId"`
	OrderID   string    `json:"orderId"`
	ExpiresIn uint      `json:"expiresIn"`
	Thumbnail Thumbnail `json:"thumbnail"`
}

func (n *BuyerDisputeExpiryNotification) Type() string { return "BuyerDisputeExpiryNotification" }

// VendorFinalizedPayment represents a notification about a purchase
// which will soon be unable to dispute.
type VendorFinalizedPaymentNotification struct {
	ID      string `json:"notificationId"`
	OrderID string `json:"orderId"`
}

func (n *VendorFinalizedPaymentNotification) Type() string {
	return "VendorFinalizedPaymentNotification"
}

// ModeratorDisputeExpiry represents a notification about an open dispute
// which will soon be expired and automatically resolved. The Type indicates
// the age of the dispute case and the CaseID references the cases caseID
// in the database schema
type ModeratorDisputeExpiryNotification struct {
	ID        string    `json:"notificationId"`
	CaseID    string    `json:"disputeCaseId"`
	ExpiresIn uint      `json:"expiresIn"`
	Thumbnail Thumbnail `json:"thumbnail"`
}

func (n *ModeratorDisputeExpiryNotification) Type() string {
	return "ModeratorDisputeExpiryNotification"
}

// AddressRequestResponseNotification represents a notification which fires
// in response to the AddressRequst message.
type AddressRequestResponseNotification struct {
	PeerID  string `json:"peerID"`
	Address string `json:"address"`
	Coin    string `json:"coin"`
}

func (n *AddressRequestResponseNotification) Type() string {
	return "AddressRequestResponseNotification"
}

// TestNotification is a test notification.
type TestNotification struct{}

func (n *TestNotification) Type() string { return "TestNotification" }

package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cpacia/openbazaar3.0/events"
	"time"
)

// NotificationRecord encapsulates one of many notifications with additional
// metadata. The actual notification is serialized as JSON so as to
// make this model suitable for the database. It may also be sent over
// the websocket API in this format.
type NotificationRecord struct {
	ID         string          `gorm:"primary_key" json:"-"`
	Timestamp  time.Time       `json:"timestamp"`
	IsRead     bool            `json:"read"`
	Serialized json.RawMessage `json:"notification"`
	Type       string          `json:"type"`
}

// NewNotificationRecord takes in a notification and returns a new NotificationRecord
// with a new ID and timestamp.
func NewNotificationRecord(notification events.TypedNotification) (*NotificationRecord, error) {
	out, err := json.MarshalIndent(notification, "", "    ")
	if err != nil {
		return nil, err
	}

	return &NotificationRecord{
		ID:         newNofificationID(),
		Timestamp:  time.Now(),
		Type:       string(notification.Type()),
		Serialized: out,
	}, nil
}

func (n *NotificationRecord) Notification() (events.TypedNotification, error) {
	notif, ok := notificationMap[n.Type]
	if !ok {
		return nil, fmt.Errorf("unknown notification type: %s", n.Type)
	}
	if err := json.Unmarshal(n.Serialized, notif); err != nil {
		return nil, err
	}
	return notif, nil
}

func newNofificationID() string {
	r := make([]byte, 20)
	rand.Read(r)
	return base64.StdEncoding.EncodeToString(r)
}

var notificationMap = map[string]events.TypedNotification{
	"OrderNotification":                  &events.OrderNotification{},
	"PaymentNotification":                &events.PaymentNotification{},
	"OrderConfirmationNotification":      &events.OrderConfirmationNotification{},
	"OrderDeclinedNotification":          &events.OrderDeclinedNotification{},
	"OrderCancelNotification":            &events.OrderCancelNotification{},
	"RefundNotification":                 &events.RefundNotification{},
	"FulfillmentNotification":            &events.FulfillmentNotification{},
	"ProcessingErrorNotification":        &events.ProcessingErrorNotification{},
	"CompletionNotification":             &events.CompletionNotification{},
	"DisputeOpenNotification":            &events.DisputeOpenNotification{},
	"DisputeUpdateNotification":          &events.DisputeUpdateNotification{},
	"DisputeCloseNotification":           &events.DisputeCloseNotification{},
	"DisputeAcceptedNotification":        &events.DisputeAcceptedNotification{},
	"FollowNotification":                 &events.FollowNotification{},
	"UnfollowNotification":               &events.UnfollowNotification{},
	"ModeratorAddNotification":           &events.ModeratorAddNotification{},
	"ModeratorRemoveNotification":        &events.ModeratorRemoveNotification{},
	"StatusNotification":                 &events.StatusNotification{},
	"ChatMessageNotification":            &events.ChatMessageNotification{},
	"ChatReadNotification":               &events.ChatReadNotification{},
	"ChatTypingNotification":             &events.ChatTypingNotification{},
	"IncomingTransactionNotification":    &events.IncomingTransactionNotification{},
	"VendorDisputeTimeoutNotification":   &events.VendorDisputeTimeoutNotification{},
	"BuyerDisputeTimeoutNotification":    &events.BuyerDisputeTimeoutNotification{},
	"BuyerDisputeExpiryNotification":     &events.BuyerDisputeExpiryNotification{},
	"VendorFinalizedPaymentNotification": &events.VendorFinalizedPaymentNotification{},
	"ModeratorDisputeExpiryNotification": &events.ModeratorDisputeExpiryNotification{},
	"TestNotification":                   &events.TestNotification{},
}

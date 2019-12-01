package models

import (
	"encoding/json"
	"time"
)

// NotificationRecord encapsulates one of many notifications with additional
// metadata. The actual notification is serialized as JSON so as to
// make this model suitable for the database.
type NotificationRecord struct {
	ID           string          `gorm:"primary_key" json:"-"`
	Timestamp    time.Time       `json:"timestamp"`
	Read         bool            `json:"read"`
	Notification json.RawMessage `json:"notification"`
}

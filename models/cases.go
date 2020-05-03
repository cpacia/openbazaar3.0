package models

import "encoding/json"

type Case struct {
	ID OrderID `gorm:"primary_key"`

	BuyerContract  json.RawMessage
	VendorContract json.RawMessage

	DisputeOpen  json.RawMessage
	DisputeClose json.RawMessage
}

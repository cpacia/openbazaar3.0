package models

import (
	iwallet "github.com/cpacia/wallet-interface"
)

// TransactionMetadata is the data model for wallet transaction
// metadata that is stored in the database. This is extra metadata
// beyond what is saved by the multiwallet.
type TransactionMetadata struct {
	Txid           iwallet.TransactionID `gorm:"primary_key"`
	PaymentAddress string
	Memo           string
	OrderID        OrderID
	Thumbnail      string
}

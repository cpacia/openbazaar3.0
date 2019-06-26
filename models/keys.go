package models

import "github.com/jinzhu/gorm"

// Key holds raw key data used by the node and stored in the
// database. The name field identifies the key and is used as
// the primary key.
type Key struct {
	gorm.Model
	Name  string `gorm:"unique;not null;UNIQUE_INDEX"`
	Value []byte
}

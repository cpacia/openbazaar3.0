package models

import "github.com/ipfs/go-cid"

type OrderID cid.Cid

type Order struct {
	ID             OrderID `gorm:"primary_key"`
	ParkedMessages [][]byte
}

package models

type OrderID string

type Order struct {
	ID             OrderID `gorm:"primary_key"`
	ParkedMessages [][]byte
}

package factory

import (
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
)

func NewPurchase() *models.Purchase {
	return &models.Purchase{
		ShipTo:       "Peter",
		Address:      "123 Spooner St.",
		City:         "Quahog",
		State:        "RI",
		PostalCode:   "90210",
		CountryCode:  pb.CountryCode_UNITED_STATES.String(),
		AddressNotes: "asdf",
		Moderator:    "",
		Items: []models.PurchaseItem{
			{
				Quantity: 1,
				Options: []models.PurchaseItemOption{
					{
						Name:  "size",
						Value: "large",
					},
					{
						Name:  "color",
						Value: "red",
					},
				},
				Shipping: models.PurchaseShippingOption{
					Name:    "usps",
					Service: "standard",
				},
				Memo: "I want it fast!",
			},
		},
		AlternateContactInfo: "peter@protonmail.com",
		PaymentCoin:          "TMCK",
	}
}

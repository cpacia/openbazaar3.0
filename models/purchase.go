package models

import iwallet "github.com/cpacia/wallet-interface"

// PurchaseItemOption is the item option selection.
type PurchaseItemOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PurchaseShippingOption is the shipping option selection.
type PurchaseShippingOption struct {
	Name    string `json:"name"`
	Service string `json:"service"`
}

// PurchaseItem is information about the item in the purchase.
type PurchaseItem struct {
	ListingHash    string                 `json:"listingHash"`
	Quantity       string                 `json:"quantity"`
	Options        []PurchaseItemOption   `json:"options"`
	Shipping       PurchaseShippingOption `json:"shipping"`
	Memo           string                 `json:"memo"`
	Coupons        []string               `json:"coupons"`
	PaymentAddress string                 `json:"paymentAddress"`
}

// Purchase contains all the information needed by the node to
// execute a purchase.
type Purchase struct {
	ShipTo               string         `json:"shipTo"`
	Address              string         `json:"address"`
	City                 string         `json:"city"`
	State                string         `json:"state"`
	PostalCode           string         `json:"postalCode"`
	CountryCode          string         `json:"countryCode"`
	AddressNotes         string         `json:"addressNotes"`
	Moderator            string         `json:"moderator"`
	Items                []PurchaseItem `json:"items"`
	AlternateContactInfo string         `json:"alternateContactInfo"`
	RefundAddress        *string        `json:"refundAddress"` //optional, can be left out of json
	PaymentCoin          string         `json:"paymentCoin"`
}

// OrderTotals represents a breakdown of the various charges of the order.
type OrderTotals struct {
	Subtotal  iwallet.Amount `json:"subtotal"`
	Shipping  iwallet.Amount `json:"shipping"`
	Discounts iwallet.Amount `json:"discounts"`
	Taxes     iwallet.Amount `json:"taxes"`
	Total     iwallet.Amount `json:"total"`
}

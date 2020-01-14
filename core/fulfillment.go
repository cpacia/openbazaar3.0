package core

import "github.com/cpacia/openbazaar3.0/models"

// FulfillOrder sends an order fulfillment to the remote peer and updates the order state.
func (n *OpenBazaarNode) FulfillOrder(orderID models.OrderID, fulfillments []models.Fulfillment, done chan struct{}) error {
	return nil
}

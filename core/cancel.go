package core

import "github.com/cpacia/openbazaar3.0/models"

// CancelOrder is called only by the buyer and sends an ORDER_CANCEL message to the vendor
// while releasing the funds from the 1 of 2 multisig back into this wallet. This can only
// be called when the order payment method is CANCELABLE and the order has not been confirmed
// or progressed any further.
//
// Note there is a possibility of a race between this function and ConfirmOrder called by
// the vendor. In such a scenario this function will return without error but we will
// later determine which person "wins" based on which transaction confirmed in the blockchain.
func (n *OpenBazaarNode) CancelOrder(orderID models.OrderID, done chan struct{}) error {
	return nil
}

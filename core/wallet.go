package core

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	iwallet "github.com/cpacia/wallet-interface"
)

// SaveTransactionMetadata saves additional metadata for a wallet transaction into the database.
// We use the provided payment address to look up if any order matches the payment address and
// use it to populate some additional metadata.
func (n *OpenBazaarNode) SaveTransactionMetadata(metadata *models.TransactionMetadata) error {
	return n.repo.DB().Update(func(tx database.Tx) error {
		var order models.Order
		err := tx.Read().Where("payment_address = ?", metadata.PaymentAddress).First(&order).Error
		if err == nil {
			metadata.OrderID = order.ID

			orderOpen, err := order.OrderOpenMessage()
			if err == nil {
				metadata.Thumbnail = orderOpen.Listings[0].Listing.Item.Images[0].Tiny
				if metadata.Memo == "" {
					metadata.Memo = orderOpen.Listings[0].Listing.Item.Title
				}
			}
		}
		return tx.Save(metadata)
	})
}

// GetTransactionMetadata loads and returns the transaction metadata from the db given the transaction ID.
func (n *OpenBazaarNode) GetTransactionMetadata(txid iwallet.TransactionID) (models.TransactionMetadata, error) {
	var metadata models.TransactionMetadata
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("txid=?", txid.String()).First(&metadata).Error
	})
	return metadata, err
}

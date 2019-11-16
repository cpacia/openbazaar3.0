package orders

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/models/factory"
	iwallet "github.com/cpacia/wallet-interface"
	"testing"
)

func TestOrderProcessor_processWalletTransaction(t *testing.T) {
	tests := []struct {
		setup    func() (*OrderProcessor, func(), error)
		tx       iwallet.Transaction
		validate func(op *OrderProcessor) error
	}{
		{
			setup: func() (*OrderProcessor, func(), error) {
				op, teardown, err := newMockOrderProcessor()
				if err != nil {
					return nil, nil, err
				}

				err = op.db.Update(func(tx database.Tx) error {
					orderOpen, _, err := factory.NewOrder()
					if err != nil {
						return err
					}
					orderOpen.Payment.Address = "abcd"
					order := models.Order{
						ID:             "1234",
						PaymentAddress: "abcd",
					}
					order.SetRole(models.RoleBuyer)
					if err := order.PutMessage(orderOpen); err != nil {
						return err
					}
					return tx.Save(&order)

				})
				if err != nil {
					return nil, nil, err
				}

				return op, teardown, nil
			},
			tx: iwallet.Transaction{
				ID: "5678",
				To: []iwallet.SpendInfo{
					{
						ID:      nil,
						Address: iwallet.NewAddress("abcd", iwallet.CtTestnetMock),
						Amount:  iwallet.NewAmount(4992221),
					},
				},
				Height: 0,
			},
			validate: func(op *OrderProcessor) error {
				var order models.Order
				err := op.db.View(func(tx database.Tx) error {
					return tx.Read().Where("id = ?", "1234").First(&order).Error
				})
				if err != nil {
					return err
				}
				txs, err := order.GetTransactions()
				if err != nil {
					return err
				}
				if len(txs) != 1 {
					return errors.New("failed to record transaction")
				}
				funded, err := order.IsFunded()
				if err != nil {
					return err
				}
				if !funded {
					return errors.New("failed to set order as funded")
				}
				sent, err := order.PaymentSentMessages()
				if err != nil {
					return err
				}
				if len(sent) != 1 {
					return errors.New("failed to payment sent message")
				}
				return nil
			},
		},
		{
			setup: func() (*OrderProcessor, func(), error) {
				op, teardown, err := newMockOrderProcessor()
				if err != nil {
					return nil, nil, err
				}

				err = op.db.Update(func(tx database.Tx) error {
					orderOpen, _, err := factory.NewOrder()
					if err != nil {
						return err
					}
					orderOpen.Payment.Address = "abcd"
					order := models.Order{
						ID:             "1234",
						PaymentAddress: "abcd",
					}
					order.SetRole(models.RoleVendor)
					if err := order.PutMessage(orderOpen); err != nil {
						return err
					}
					return tx.Save(&order)

				})
				if err != nil {
					return nil, nil, err
				}

				return op, teardown, nil
			},
			tx: iwallet.Transaction{
				ID: "5678",
				To: []iwallet.SpendInfo{
					{
						ID:      nil,
						Address: iwallet.NewAddress("abcd", iwallet.CtTestnetMock),
						Amount:  iwallet.NewAmount(4992221),
					},
				},
				Height: 0,
			},
			validate: func(op *OrderProcessor) error {
				var order models.Order
				err := op.db.View(func(tx database.Tx) error {
					return tx.Read().Where("id = ?", "1234").First(&order).Error
				})
				if err != nil {
					return err
				}
				txs, err := order.GetTransactions()
				if err != nil {
					return err
				}
				if len(txs) != 1 {
					return errors.New("failed to record transaction")
				}
				funded, err := order.IsFunded()
				if err != nil {
					return err
				}
				if !funded {
					return errors.New("failed to set order as funded")
				}
				return nil
			},
		},
		{
			setup: func() (*OrderProcessor, func(), error) {
				op, teardown, err := newMockOrderProcessor()
				if err != nil {
					return nil, nil, err
				}

				err = op.db.Update(func(tx database.Tx) error {
					orderOpen, _, err := factory.NewOrder()
					if err != nil {
						return err
					}
					orderOpen.Payment.Address = "abcd"
					order := models.Order{
						ID:             "1234",
						PaymentAddress: "abcd",
					}
					if err := order.PutMessage(orderOpen); err != nil {
						return err
					}
					return tx.Save(&order)

				})
				if err != nil {
					return nil, nil, err
				}

				return op, teardown, nil
			},
			tx: iwallet.Transaction{
				ID: "5678",
				From: []iwallet.SpendInfo{
					{
						ID:      nil,
						Address: iwallet.NewAddress("abcd", iwallet.CtTestnetMock),
						Amount:  iwallet.NewAmount(4992221),
					},
				},
				Height: 0,
			},
			validate: func(op *OrderProcessor) error {
				var order models.Order
				err := op.db.View(func(tx database.Tx) error {
					return tx.Read().Where("id = ?", "1234").First(&order).Error
				})
				if err != nil {
					return err
				}
				txs, err := order.GetTransactions()
				if err != nil {
					return err
				}
				if len(txs) != 1 {
					return errors.New("failed to record transaction")
				}
				return nil
			},
		},
	}

	for i, test := range tests {
		op, teardown, err := test.setup()
		if err != nil {
			t.Errorf("Test %d setup failed: %s", i, err)
			continue
		}
		op.processWalletTransaction(test.tx)
		if err := test.validate(op); err != nil {
			t.Errorf("Test %d validation failed: %s", i, err)
		}
		teardown()
	}
}

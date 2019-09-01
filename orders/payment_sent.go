package orders

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
)

func (op *OrderProcessor) handlePaymentSentMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	payment := new(pb.PaymentSent)
	if err := ptypes.UnmarshalAny(message.Message, payment); err != nil {
		return nil, err
	}

	orderOpen, err := order.OrderOpenMessage()
	if models.IsMessageNotExistError(err) {
		return nil, order.ParkMessage(message)
	}
	if err != nil {
		return nil, err
	}

	err = order.PutMessage(payment)
	if models.IsDuplicateTransactionError(err) {
		return nil, nil
	}

	wallet, err := op.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}

	txs, err := order.GetTransactions()
	if err != nil && !models.IsDuplicateTransactionError(err) {
		return nil, err
	}

	for _, tx := range txs {
		if tx.ID.String() == payment.TransactionID {
			log.Debugf("Received PAYMENT_SENT message for order %s but already know about transaction", order.ID)
			return nil, nil
		}
	}

	// If this fails it's OK as the processor's unfunded order checking loop will
	// retry at it's next interval.
	tx, err := wallet.GetTransaction(iwallet.TransactionID(payment.TransactionID))
	if err == nil {
		for _, to := range tx.To {
			if to.Address.String() == order.PaymentAddress {
				if err := op.handleIncomingPayment(dbtx, order, to, tx); err != nil {
					return nil, err
				}
			}
		}
	}

	log.Infof("Received PAYMENT_SENT message for order %s", order.ID)

	event := &events.PaymentSentNotification{
		OrderID: order.ID.String(),
		Txid:    payment.TransactionID,
	}
	return event, nil
}

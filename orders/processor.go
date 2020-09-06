package orders

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/jinzhu/gorm"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/op/go-logging"
	"sort"
	"time"
)

const rescanTransactionsInterval = time.Minute

var (
	log                  = logging.MustGetLogger("ORDR")
	ErrChangedMessage    = errors.New("different duplicate message")
	ErrUnexpectedMessage = errors.New("unexpected message")
)

// Config holds the objects needed to instantiate a new OrderProcessor.
type Config struct {
	Identity             peer.ID
	Db                   database.Database
	IdentityPrivateKey   crypto.PrivKey
	EscrowPrivateKey     *btcec.PrivateKey
	Messenger            *net.Messenger
	Multiwallet          multiwallet.Multiwallet
	ExchangeRateProvider *wallet.ExchangeRateProvider
	EventBus             events.Bus
	CalcCIDFunc          func(file []byte) (cid.Cid, error)
}

// OrderProcessor is used to deterministically process orders.
type OrderProcessor struct {
	identity           peer.ID
	identityPrivateKey crypto.PrivKey
	db                 database.Database
	messenger          *net.Messenger
	multiwallet        multiwallet.Multiwallet
	escrowPrivateKey   *btcec.PrivateKey
	erp                *wallet.ExchangeRateProvider
	bus                events.Bus
	calcCIDFunc        func(file []byte) (cid.Cid, error)
	shutdown           chan struct{}
}

// NewOrderProcessor initializes and returns a new OrderProcessor
func NewOrderProcessor(cfg *Config) *OrderProcessor {
	return &OrderProcessor{
		identity:           cfg.Identity,
		identityPrivateKey: cfg.IdentityPrivateKey,
		db:                 cfg.Db,
		messenger:          cfg.Messenger,
		multiwallet:        cfg.Multiwallet,
		escrowPrivateKey:   cfg.EscrowPrivateKey,
		erp:                cfg.ExchangeRateProvider,
		bus:                cfg.EventBus,
		calcCIDFunc:        cfg.CalcCIDFunc,
		shutdown:           make(chan struct{}),
	}
}

// Start begins listening for transactions from the wallets that pertain to our
// orders. When we find one we record the payment.
func (op *OrderProcessor) Start() {
	go op.checkForMorePayments()
	ticker := time.NewTicker(rescanTransactionsInterval)
	for _, wallet := range op.multiwallet {
		go func(w iwallet.Wallet) {
			sub := w.SubscribeTransactions()
			for {
				select {
				case tx := <-sub:
					op.processWalletTransaction(tx)
				case <-ticker.C:
					op.checkForMorePayments()
				case <-op.shutdown:
					return
				}
			}
		}(wallet)
	}
}

// Stop shuts down the processor.
func (op *OrderProcessor) Stop() {
	close(op.shutdown)
}

// ProcessMessage is the main handler for the OrderProcessor. It ingests a new message
// loads the corresponding order from the database, passes the message off to the appropriate
// handler for processing, then saves the updated state back into the database.
// Any messages that arrive out of order are saved in the database as a parked message which
// will allow for future processing. The same is said for messages that error.
//
// The end result of this process is if the buyer and vendor pass in the same set of messages
// into this function, regardless of order, the exact same state should be calculated for
// both nodes.
//
// If the processing of the message triggers an event to emitted onto the bus, the event is
// returned.
func (op *OrderProcessor) ProcessMessage(dbtx database.Tx, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	// Load the order if it exists.
	var (
		order models.Order
		event interface{}
		err   error
	)
	err = dbtx.Read().Where("id = ?", message.OrderID).First(&order).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return nil, err
	} else if gorm.IsRecordNotFoundError(err) && message.MessageType != npb.OrderMessage_ORDER_OPEN {
		// Order does not exist in the DB and the message type is not an order open. This can happen
		// in the case where we download offline messages out of order. In this case we will park
		// the message so that we can try again later if we receive other messages.
		log.Warningf("Received %s message from peer %d for an order that does not exist yet", message.MessageType, peer.Pretty())
		order.ID = models.OrderID(message.OrderID)
		if err := order.ParkMessage(message); err != nil {
			return nil, err
		}
		return nil, dbtx.Save(&order)
	}

	orderCopy := order
	event, err = op.processMessage(dbtx, &order, peer, message)
	if err != nil {
		log.Errorf("Error processing order message for order %s: %s", order.ID.String(), err)
		if err := orderCopy.PutErrorMessage(message); err != nil {
			return nil, dbtx.Save(&orderCopy)
		}
	}

	parkedMessages, err := order.GetParkedMessages()
	if err != nil {
		return nil, err
	}

	sort.Slice(parkedMessages, func(i, j int) bool {
		return parkedMessages[i].MessageType < parkedMessages[j].MessageType
	})

	for _, parked := range parkedMessages {
		if proto.Equal(parked, message) {
			continue
		}
		_, err = op.processMessage(dbtx, &order, peer, parked)
		if err != nil {
			log.Errorf("Error processing parked message for order %s: %s", order.ID.String(), err)
			if err := order.PutErrorMessage(message); err != nil {
				log.Errorf("Error saving errored message for order %s: %s", order.ID.String(), err)
			}
		}
	}

	// Save changes to the database.
	return event, dbtx.Save(&order)
}

// ProcessACK loads the order from the database and sets the ACK for the message type.
func (op *OrderProcessor) ProcessACK(tx database.Tx, om *models.OutgoingMessage) error {
	message := new(npb.Message)
	if err := proto.Unmarshal(om.SerializedMessage, message); err != nil {
		return err
	}

	orderMessage := new(npb.OrderMessage)
	if err := ptypes.UnmarshalAny(message.Payload, orderMessage); err != nil {
		return err
	}

	var key string
	switch orderMessage.MessageType {
	case npb.OrderMessage_ORDER_OPEN:
		key = "order_open_acked"
	case npb.OrderMessage_ORDER_REJECT:
		key = "order_reject_acked"
	case npb.OrderMessage_ORDER_CANCEL:
		key = "order_cancel_acked"
	case npb.OrderMessage_ORDER_CONFIRMATION:
		key = "order_confirmation_acked"
	case npb.OrderMessage_ORDER_FULFILLMENT:
		key = "order_fulfillment_acked"
	case npb.OrderMessage_ORDER_COMPLETE:
		key = "order_complete_acked"
	case npb.OrderMessage_DISPUTE_OPEN:
		if message.MessageType == npb.Message_DISPUTE {
			key = "dispute_open_moderator_acked"
		} else {
			key = "dispute_open_other_party_acked"
		}
	case npb.OrderMessage_DISPUTE_UPDATE:
		key = "dispute_update_acked"
	case npb.OrderMessage_DISPUTE_CLOSE:
		key = "dispute_closed_acked"
	case npb.OrderMessage_REFUND:
		key = "refund_acked"
	case npb.OrderMessage_PAYMENT_SENT:
		key = "payment_sent_acked"
	case npb.OrderMessage_RATING_SIGNATURES:
		key = "rating_signatures_acked"
	case npb.OrderMessage_PAYMENT_FINALIZED:
		key = "payment_finalized_acked"
	default:
		return fmt.Errorf("unknown order message type")
	}
	return tx.Update(key, true, map[string]interface{}{"id = ?": orderMessage.OrderID}, &models.Order{})
}

// processMessage passes the message off to the appropriate handler.
func (op *OrderProcessor) processMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (event interface{}, err error) {
	err = verifyOrderMessageSignature(peer, message)
	if err != nil {
		return nil, err
	}
	switch message.MessageType {
	case npb.OrderMessage_ORDER_OPEN:
		event, err = op.processOrderOpenMessage(dbtx, order, peer, message)
	case npb.OrderMessage_PAYMENT_SENT:
		event, err = op.processPaymentSentMessage(dbtx, order, peer, message)
	case npb.OrderMessage_RATING_SIGNATURES:
		event, err = op.processRatingSignaturesMessage(dbtx, order, peer, message)
	case npb.OrderMessage_ORDER_REJECT:
		event, err = op.processOrderRejectMessage(dbtx, order, peer, message)
	case npb.OrderMessage_ORDER_CONFIRMATION:
		event, err = op.processOrderConfirmationMessage(dbtx, order, peer, message)
	case npb.OrderMessage_ORDER_CANCEL:
		event, err = op.processOrderCancelMessage(dbtx, order, peer, message)
	case npb.OrderMessage_REFUND:
		event, err = op.processRefundMessage(dbtx, order, peer, message)
	case npb.OrderMessage_ORDER_FULFILLMENT:
		event, err = op.processOrderFulfillmentMessage(dbtx, order, peer, message)
	case npb.OrderMessage_ORDER_COMPLETE:
		event, err = op.processOrderCompleteMessage(dbtx, order, peer, message)
	case npb.OrderMessage_DISPUTE_OPEN:
		event, err = op.processDisputeOpenMessage(dbtx, order, peer, message)

	default:
		return nil, errors.New("unknown order message type")
	}
	return event, err
}

// isDuplicate checks the serialization of the passed in message against the
// passed in serialization and returns true if they match.
func isDuplicate(message proto.Message, serialized []byte) (bool, error) {
	m := jsonpb.Marshaler{
		EmitDefaults: true,
		Indent:       "    ",
	}

	ser, err := m.MarshalToString(message)
	if err != nil {
		return false, err
	}

	return bytes.Equal([]byte(ser), serialized), nil
}

func verifyOrderMessageSignature(peer peer.ID, message *npb.OrderMessage) error {
	peerPubkey, err := peer.ExtractPublicKey()

	msgCpy := *message
	msgCpy.Signature = nil

	ser, err := proto.Marshal(&msgCpy)
	if err != nil {
		return err
	}

	valid, err := peerPubkey.Verify(ser, message.Signature)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("invalid signature")
	}
	return nil
}

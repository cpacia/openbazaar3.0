package notifications

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/op/go-logging"
	"time"
)

var log = logging.MustGetLogger("notif")

type notificationWrapper struct {
	Notification interface{} `json:"notification"`
}

type chatMessageWrapper struct {
	ChatMessage interface{} `json:"chatMessage"`
}

type messageReadWrapper struct {
	MessageRead interface{} `json:"messageRead"`
}

type messageTypingWrapper struct {
	MessageTyping interface{} `json:"messageTyping"`
}

type walletWrapper struct {
	Wallet interface{} `json:"wallet"`
}

// Notifier manages translating events into notifications and
// sending them to websockets.
type Notifier struct {
	notifyFunc func(interface{}) error
	bus        events.Bus
	db         database.Database
	shutdown   chan struct{}
}

// NewNotifier returns a new notifer.
func NewNotifier(bus events.Bus, db database.Database, notifyFunc func(interface{}) error) *Notifier {
	return &Notifier{
		bus:        bus,
		db:         db,
		notifyFunc: notifyFunc,
		shutdown:   make(chan struct{}),
	}
}

// Start will start up the notifier. This should use it's own goroutine.
func (n *Notifier) Start() {
	notifications := []interface{}{
		&events.NewOrder{},
		&events.OrderFunded{},
		&events.OrderPaymentReceived{},
		&events.OrderConfirmation{},
		&events.OrderDeclined{},
		&events.OrderCancel{},
		&events.Refund{},
		&events.OrderFulfillment{},
		&events.OrderCompletion{},
		&events.DisputeOpen{},
		&events.DisputeUpdate{},
		&events.DisputeClose{},
		&events.DisputeAccepted{},
		&events.VendorFinalizedPayment{},
		&events.Follow{},
		&events.Unfollow{},
	}

	notificationSub, err := n.bus.Subscribe(notifications)
	if err != nil {
		log.Errorf("Error subscribing to events: %s", err)
	}

	chats := []interface{}{
		&events.ChatMessage{},
		&events.ChatRead{},
		&events.ChatTyping{},
	}

	chatSub, err := n.bus.Subscribe(chats)
	if err != nil {
		log.Errorf("Error subscribing to events: %s", err)
	}

	for {
		select {
		case event := <-notificationSub.Out():
			id := convertToNotification(event)

			out, err := json.MarshalIndent(event, "", "    ")
			if err != nil {
				log.Errorf("Error saving notification to the database: %s", err)
				continue
			}

			err = n.db.Update(func(tx database.Tx) error {
				return tx.Save(&models.NotificationRecord{
					ID:           id,
					Timestamp:    time.Now(),
					Read:         false,
					Notification: out,
				})
			})
			if err != nil {
				log.Errorf("Error saving notification to the database: %s", err)
				continue
			}

			if err := n.notifyFunc(notificationWrapper{event}); err != nil {
				log.Errorf("Error sending notification: %s", err)
			}
		case event := <-chatSub.Out():
			var i interface{}
			switch event.(type) {
			case *events.ChatMessage:
				i = chatMessageWrapper{event}
			case *events.ChatRead:
				i = messageReadWrapper{event}
			case *events.ChatTyping:
				i = messageTypingWrapper{event}
			}

			if err := n.notifyFunc(i); err != nil {
				log.Errorf("Error sending notification: %s", err)
			}
		case <-n.shutdown:
			return
		}
	}
}

// Stop shuts down the notifier.
func (n *Notifier) Stop() {
	close(n.shutdown)
}

func convertToNotification(event interface{}) string {
	r := make([]byte, 20)
	rand.Read(r)
	id := hex.EncodeToString(r)

	switch e := event.(type) {
	case *events.NewOrder:
		e.Typ = "NewOrder"
		e.ID = id
	case *events.OrderFunded:
		e.Typ = "OrderFunded"
		e.ID = id
	case *events.OrderPaymentReceived:
		e.Typ = "OrderPaymentReceived"
		e.ID = id
	case *events.OrderConfirmation:
		e.Typ = "OrderConfirmation"
		e.ID = id
	case *events.OrderDeclined:
		e.Typ = "OrderDeclined"
		e.ID = id
	case *events.OrderCancel:
		e.Typ = "OrderCancel"
		e.ID = id
	case *events.Refund:
		e.Typ = "Refund"
		e.ID = id
	case *events.OrderFulfillment:
		e.Typ = "OrderFulfillment"
		e.ID = id
	case *events.OrderCompletion:
		e.Typ = "OrderCompletion"
		e.ID = id
	case *events.DisputeOpen:
		e.Typ = "DisputeOpen"
		e.ID = id
	case *events.DisputeUpdate:
		e.Typ = "DisputeUpdate"
		e.ID = id
	case *events.DisputeClose:
		e.Typ = "DisputeClose"
		e.ID = id
	case *events.DisputeAccepted:
		e.Typ = "DisputeAccepted"
		e.ID = id
	case *events.VendorFinalizedPayment:
		e.Typ = "VendorFinalizedPayment"
		e.ID = id
	case *events.Follow:
		e.Typ = "Follow"
		e.ID = id
	case *events.Unfollow:
		e.Typ = "Unfollow"
		e.ID = id
	}

	return id
}

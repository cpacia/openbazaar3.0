package notifications

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/repo"
	"testing"
	"time"
)

func TestNotifier(t *testing.T) {
	bus := events.NewBus()
	db, err := repo.MockDB()
	if err != nil {
		t.Fatal(err)
	}
	out := make(chan interface{})
	notifFunc := func(i interface{}) error {
		out <- i
		return nil
	}

	sub, err := bus.Subscribe(&notifierStarted{})
	if err != nil {
		t.Fatal(err)
	}

	notifier := NewNotifier(bus, db, notifFunc)
	go notifier.Start()
	defer notifier.Stop()

	select {
	case <-sub.Out():
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on channel")
	}

	tests := []interface{}{
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

	for _, test := range tests {

		bus.Emit(test)

		select {
		case n1 := <-out:
			wrapper, ok := n1.(notificationWrapper)
			if !ok {
				t.Fatal("Invalid notification type")
			}

			if wrapper.Notification != test {
				t.Errorf("Failed to return expected event")
			}
		case <-time.After(time.Second * 10):
			t.Fatal("Timed out waiting on channel")
		}
	}

	test := &events.ChatMessage{}
	bus.Emit(test)

	select {
	case n1 := <-out:
		_, ok := n1.(chatMessageWrapper)
		if !ok {
			t.Fatal("Invalid notification type")
		}
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on channel")
	}

	test2 := &events.ChatTyping{}
	bus.Emit(test2)

	select {
	case n1 := <-out:
		_, ok := n1.(messageTypingWrapper)
		if !ok {
			t.Fatal("Invalid notification type")
		}
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on channel")
	}

	test3 := &events.ChatRead{}
	bus.Emit(test3)

	select {
	case n1 := <-out:
		_, ok := n1.(messageReadWrapper)
		if !ok {
			t.Fatal("Invalid notification type")
		}
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out waiting on channel")
	}
}

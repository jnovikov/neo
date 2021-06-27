package pubsub

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewSubscription(t *testing.T) {
	const channel = "test"
	sub := NewSubscription(channel, nil)
	if sub.GetChannel() != channel {
		t.Errorf("NewSubscription(): invalid channel, expected %s, got %s", channel, sub.GetChannel())
	}

	newSub := NewSubscription(channel, nil)
	if sub.GetID() == newSub.GetID() {
		t.Errorf("NewSubscription(): subscription ID is not random")
	}
}

func Test_subscription_Push(t *testing.T) {
	sub := &subscription{
		id:      "test",
		channel: "test",
		queue:   nil,
		notify:  make(chan struct{}),
		h:       nil,
	}
	sub.Push("test")
	if need := []interface{}{"test"}; !reflect.DeepEqual(sub.queue, need) {
		t.Errorf("Push(): invalid queue state: expected %+v, got %+v", need, sub.queue)
	}
}

func Test_subscription_Run(t *testing.T) {
	var received string
	signal := make(chan struct{})
	handler := func(msg interface{}) error {
		cmd, ok := msg.(string)
		if !ok {
			t.Errorf("Invalid message passed to handler: %v", msg)
		}
		received = cmd
		close(signal)
		return nil
	}
	sub := &subscription{
		id:      "test",
		channel: "test",
		queue:   nil,
		notify:  make(chan struct{}, 1),
		h:       handler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sub.Run(ctx)

	const msg = "message"
	sub.Push(msg)

	select {
	case <-signal:
		break
	case <-time.After(time.Millisecond * 100):
		t.Errorf("Handler was not called in time")
	}

	if received != msg {
		t.Errorf("Received incorrect command: expected = %v, actual = %v", msg, received)
	}
}

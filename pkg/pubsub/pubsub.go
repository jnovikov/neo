package pubsub

import (
	"sync"
)

type PubSub interface {
	Publish(string, interface{})
	Subscribe(string, MessageHandler) Subscription
	Unsubscribe(Subscription)
}

type pubsub struct {
	subs map[string]map[string]Subscription
	mu   sync.RWMutex
}

func NewPubSub() PubSub {
	return &pubsub{subs: make(map[string]map[string]Subscription)}
}

func (ps *pubsub) Publish(channel string, msg interface{}) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, sub := range ps.subs[channel] {
		sub.Push(msg)
	}
}

func (ps *pubsub) Subscribe(channel string, onMsg MessageHandler) Subscription {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	sub := NewSubscription(channel, onMsg)
	if _, ok := ps.subs[channel]; !ok {
		ps.subs[channel] = make(map[string]Subscription, 1)
	}
	ps.subs[channel][sub.GetID()] = sub
	return sub
}

func (ps *pubsub) Unsubscribe(sub Subscription) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	id, channel := sub.GetID(), sub.GetChannel()
	if chSubs, ok := ps.subs[channel]; ok {
		delete(chSubs, id)
	}
}

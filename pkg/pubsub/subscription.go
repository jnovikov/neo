package pubsub

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/sirupsen/logrus"
)

type MessageHandler func(interface{}) error

type Subscription interface {
	Run(ctx context.Context)
	Push(interface{})
	GetID() string
	GetChannel() string
}

type subscription struct {
	id      string
	channel string
	queue   []interface{}
	mu      sync.Mutex
	notify  chan struct{}
	h       MessageHandler
}

func NewSubscription(channel string, onMsg MessageHandler) Subscription {
	id := uuid.NewString()
	return &subscription{
		id:      id,
		channel: channel,
		queue:   nil,
		notify:  make(chan struct{}, 1),
		h:       onMsg,
	}
}

func (s *subscription) Run(ctx context.Context) {
	for {
		select {
		case _, ok := <-s.notify:
			if !ok {
				return
			}
			for {
				s.mu.Lock()
				if len(s.queue) == 0 {
					s.mu.Unlock()
					break
				}
				cmd := s.queue[0]
				s.queue = s.queue[1:]
				s.mu.Unlock()

				logrus.Debugf("Handling message %v in subscription %s", cmd, s.id)
				if err := s.h(cmd); err != nil {
					logrus.Errorf("Error in subscription message handler: %v", err)
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *subscription) Push(msg interface{}) {
	s.mu.Lock()
	s.queue = append(s.queue, msg)
	s.mu.Unlock()

	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *subscription) GetID() string {
	return s.id
}

func (s *subscription) GetChannel() string {
	return s.channel
}

package queue

import (
	"context"
	"errors"
)

var (
	ErrQueueFull = errors.New("queue channel is full")
)

type Factory func(maxJobs int) Queue

type Queue interface {
	Start(context.Context)
	Add(Task) error
	Stop()
	Results() <-chan *Output
}

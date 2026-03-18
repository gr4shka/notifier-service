package notifierservice

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Message struct {
	ID      string
	Payload string
}

type ExternalClient interface {
	Post(ctx context.Context, msg Message) (statusCode int, err error)
}

type Notifier struct {
	client  ExternalClient
	jobs    chan Message
	workers int
	limiter *time.Ticker
	wg      sync.WaitGroup
	closed  atomic.Bool
	stats   Stats
}

func NewNotifier(client ExternalClient, worker int, rate int) *Notifier {
	n := &Notifier{
		client:  client,
		jobs:    make(chan Message, 1000),
		workers: worker,
		limiter: time.NewTicker(time.Second / time.Duration(rate)),
	}

	ctx := context.Background()
	n.startWorkers(ctx)

	return n
}

func (n *Notifier) Send(msg Message) error {
	if n.closed.Load() {
		return errors.New("notifier closed")
	}

	n.jobs <- msg
	return nil
}

func (n *Notifier) Close() {
	n.closed.Store(true)
	close(n.jobs)
	n.wg.Wait()
	n.limiter.Stop()
}

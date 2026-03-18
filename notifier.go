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
	cancel  context.CancelFunc
	mu      sync.Mutex
	stats   Stats
}

func NewNotifier(client ExternalClient, worker int, rate int) *Notifier {
	ctx, cancel := context.WithCancel(context.Background())
	n := &Notifier{
		client:  client,
		jobs:    make(chan Message, 1000),
		workers: worker,
		limiter: time.NewTicker(time.Second / time.Duration(rate)),
		cancel:  cancel,
	}

	n.startWorkers(ctx)
	return n
}

func (n *Notifier) Send(ctx context.Context, msg Message) error {
	if n.closed.Load() {
		return errors.New("notifier closed")
	}

	select {
	case n.jobs <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Notifier) Close() {
	n.closed.Store(true)
	n.cancel()
	close(n.jobs)
	n.wg.Wait()
	n.limiter.Stop()
}

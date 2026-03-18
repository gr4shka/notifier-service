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
	cancel  context.CancelFunc
	closed  atomic.Bool
	ctx     context.Context
	once    sync.Once
	stats   Stats
}

func NewNotifier(client ExternalClient, worker int, rate int) *Notifier {
	if worker <= 0 {
		worker = 1
	}

	if rate < 1 {
		rate = 1
	}

	if rate > 1_000_000_000 {
		rate = 1_000_000_00
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &Notifier{
		client:  client,
		jobs:    make(chan Message, 1000),
		workers: worker,
		limiter: time.NewTicker(time.Second / time.Duration(rate)),
		cancel:  cancel,
		ctx:     ctx,
	}

	n.startWorkers(ctx)
	return n
}

func (n *Notifier) Send(ctx context.Context, msg Message) error {
	if n.closed.Load() {
		return errors.New("notifier is closed")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-n.ctx.Done():
		return errors.New("notifier is closed")
	case n.jobs <- msg:
		return nil
	}
}

func (n *Notifier) Close() {
	n.once.Do(func() {
		n.closed.Store(true)
		n.cancel()
		n.wg.Wait()
		n.limiter.Stop()
	})
}

func (n *Notifier) GetStats() Stats {
	return n.stats.Snapshot()
}

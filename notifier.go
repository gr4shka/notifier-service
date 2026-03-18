package notifierservice

import (
	"context"
	"sync"
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
	once    sync.Once
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
	select {
	case n.jobs <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Notifier) Close() {
	n.once.Do(func() {
		n.cancel()
		n.wg.Wait()
		n.limiter.Stop()
	})
}

func (n *Notifier) GetStats() Stats {
	return n.stats.Snapshot()
}

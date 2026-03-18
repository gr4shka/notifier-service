package notifierservice

import (
	"context"
	"time"
)

func (n *Notifier) startWorkers(ctx context.Context) {
	for i := 0; i < n.workers; i++ {
		n.wg.Add(1)
		go n.workerLoop(ctx)
	}
}

func (n *Notifier) workerLoop(ctx context.Context) {
	defer n.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-n.jobs:
			if !ok {
				return
			}

			select {
			case <-n.limiter.C:
				n.processMessage(ctx, msg)
			case <-ctx.Done():
				return
			}
		}
	}
}

func (n *Notifier) processMessage(ctx context.Context, msg Message) {
	maxRetries := 3
	backoffBase := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			n.stats.IncFailed()
			return
		default:
		}

		status, err := n.client.Post(ctx, msg)

		if err != nil {
			n.stats.IncFailed()
			return
		}

		if status == 200 {
			n.stats.IncSent()
			return
		}

		if status == 429 {
			if attempt < maxRetries-1 {
				n.stats.IncRetries()

				select {
				case <-time.After(backoffBase * (1 << attempt)):
				case <-ctx.Done():
					n.stats.IncFailed()
					return
				}

				continue
			}

			n.stats.IncFailed()
			return
		}

		n.stats.IncFailed()
		return
	}
}

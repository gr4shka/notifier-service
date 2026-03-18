package notifierservice

import (
	"context"
	"errors"
	"sync"
	"time"
)

type mockClient struct {
	mu         sync.Mutex
	callCount  int
	shouldFail bool
	return429  bool
	delay      time.Duration
}

func (m *mockClient) Post(ctx context.Context, msg Message) (statusCode int, err error) {
	m.mu.Lock()
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.return429 {
		return 429, nil
	}

	if m.shouldFail {
		return 500, errors.New("internal error")
	}

	return 200, nil
}

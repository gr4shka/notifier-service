package notifierservice

import (
	"context"
	"errors"
	"sync"
	"testing"
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

func TestBasicSend(t *testing.T) {
	client := &mockClient{}
	notifier := NewNotifier(client, 1, 100)
	defer notifier.Close()

	ctx := context.Background()
	err := notifier.Send(ctx, Message{ID: "1", Payload: "test"})

	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Sent != 1 {
		t.Errorf("Expected Sent=1, got %d", stats.Sent)
	}
}

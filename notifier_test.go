package notifierservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockClient struct {
	mu        sync.Mutex
	callCount int
	maxFail   int
	failMode  string
	delay     time.Duration
}

func (m *mockClient) Post(ctx context.Context, msg Message) (statusCode int, err error) {
	m.mu.Lock()
	m.callCount++
	callNum := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if callNum <= m.maxFail {
		if m.failMode == "429" {
			return 429, nil
		}
		return 500, errors.New("error")
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

func TestSendAfterClose(t *testing.T) {
	client := &mockClient{}
	notifier := NewNotifier(client, 1, 10)
	notifier.Close()

	ctx := context.Background()
	err := notifier.Send(ctx, Message{ID: "1", Payload: "test"})

	if err == nil {
		t.Error("Expected error after Close")
	}
}

func TestRetry429(t *testing.T) {
	client := &mockClient{maxFail: 2, failMode: "429"}

	notifier := NewNotifier(client, 1, 100)
	defer notifier.Close()

	ctx := context.Background()
	notifier.Send(ctx, Message{ID: "1", Payload: "test"})

	time.Sleep(800 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Sent != 1 {
		t.Errorf("Expected Sent=1, got %d", stats.Sent)
	}
	if stats.Retries != 2 {
		t.Errorf("Expected Retries=2, got %d", stats.Retries)
	}
}

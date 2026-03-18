package notifierservice

import (
	"context"
	"errors"
	"fmt"
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

func TestMaxRetriesExceeded(t *testing.T) {
	client := &mockClient{maxFail: 10, failMode: "429"}

	notifier := NewNotifier(client, 1, 100)
	defer notifier.Close()

	ctx := context.Background()
	notifier.Send(ctx, Message{ID: "1", Payload: "test"})

	time.Sleep(800 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Sent != 0 {
		t.Errorf("Expected Sent=0, got %d", stats.Sent)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", stats.Failed)
	}
}

func TestClientError(t *testing.T) {
	client := &mockClient{maxFail: 10, failMode: "error"}

	notifier := NewNotifier(client, 1, 10)
	defer notifier.Close()

	ctx := context.Background()
	notifier.Send(ctx, Message{ID: "1", Payload: "test"})

	time.Sleep(800 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", stats.Failed)
	}
}

func TestConcurrentSends(t *testing.T) {
	client := &mockClient{}
	notifier := NewNotifier(client, 5, 1000)
	defer notifier.Close()

	numMessages := 50
	var wg sync.WaitGroup

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			notifier.Send(ctx, Message{
				ID:      fmt.Sprintf("msg-%d", id),
				Payload: "test",
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Sent != int64(numMessages) {
		t.Errorf("Expected Sent=%d, got %d", numMessages, stats.Sent)
	}
}

func TestRateLimiting(t *testing.T) {
	client := &mockClient{}
	notifier := NewNotifier(client, 1, 10)
	defer notifier.Close()

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		notifier.Send(ctx, Message{ID: fmt.Sprintf("msg-%d", i), Payload: "test"})
	}

	time.Sleep(1200 * time.Millisecond)

	stats := notifier.GetStats()
	if stats.Sent != 10 {
		t.Errorf("Expected Sent=10, got %d", stats.Sent)
	}
}

func TestGracefulShutdown(t *testing.T) {
	client := &mockClient{delay: 50 * time.Millisecond}
	notifier := NewNotifier(client, 2, 100)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		notifier.Send(ctx, Message{ID: fmt.Sprintf("msg-%d", i), Payload: "test"})
	}

	time.Sleep(300 * time.Millisecond)

	notifier.Close()

	stats := notifier.GetStats()
	totalProcessed := stats.Sent + stats.Failed

	if totalProcessed != 5 {
		t.Errorf("Expected 5 total processed, got %d", totalProcessed)
	}
}

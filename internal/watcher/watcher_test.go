package watcher

import (
	"sync"
	"testing"
	"time"
)

func TestWatcher_Subscription(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	var received Event
	w.SubscribeFunc(func(e Event) {
		received = e
		wg.Done()
	})

	w.ForceReload("/test/path")

	// Wait with timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	if received.Path != "/test/path" {
		t.Errorf("expected /test/path, got %s", received.Path)
	}
	if received.Type != EventWrite {
		t.Errorf("expected WRITE event, got %s", received.Type)
	}
}

func TestWatcher_Debounce(t *testing.T) {
	// We can't easily test internal handleEvent debouncing without complex mocking
	// but we can check the default state.
	w, _ := New()
	if w.debounceDur != 50*time.Millisecond {
		t.Errorf("expected 50ms debounce, got %v", w.debounceDur)
	}
}

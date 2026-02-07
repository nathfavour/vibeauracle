package reactor

import (
	"sync"
	"testing"
	"time"
)

func TestReactor_Submit(t *testing.T) {
	r := New()
	defer r.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	var result string
	r.Submit(func() interface{} {
		return "finished"
	}, func(res interface{}) {
		result = res.(string)
		wg.Done()
	})

	// Wait with timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		if result != "finished" {
			t.Errorf("expected finished, got %s", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task")
	}
}

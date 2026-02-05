package reactor

import (
	"context"
	"sync"
)

// Task represents a heavy computation that should not block the UI
type Task func() interface{}

// Reactor manages background processing for the TUI
type Reactor struct {
	tasks chan queuedTask
	ctx   context.Context
	cancel context.CancelFunc
	wg    sync.WaitGroup
}

type queuedTask struct {
	fn Task
	cb func(interface{})
}

func New() *Reactor {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Reactor{
		tasks:  make(chan queuedTask, 100),
		ctx:    ctx,
		cancel: cancel,
	}
	r.start()
	return r
}

func (r *Reactor) start() {
	// Start a small pool of workers
	for i := 0; i < 4; i++ {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			for {
				select {
				case <-r.ctx.Done():
					return
				case t := <-r.tasks:
					result := t.fn()
					if t.cb != nil {
						t.cb(result)
					}
				}
			}
		}()
	}
}

// Submit offloads a task to the background pool
func (r *Reactor) Submit(fn Task, cb func(interface{})) {
	select {
	case r.tasks <- queuedTask{fn: fn, cb: cb}:
	default:
		// Drop task if queue is full to avoid deadlock
	}
}

func (r *Reactor) Stop() {
	r.cancel()
	r.wg.Wait()
}

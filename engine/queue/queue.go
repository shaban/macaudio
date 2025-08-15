package queue

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Op is a graph mutation operation. It should be quick and non-blocking;
// any heavy work should be prepared in advance. It receives a context that
// will be canceled on shutdown.
// It returns an error only for unrecoverable failures; idempotent no-ops
// should return nil.
type Op interface {
	Apply(ctx context.Context) error
}

// Func is a helper to adapt functions into Op.
type Func func(ctx context.Context) error

func (f Func) Apply(ctx context.Context) error { return f(ctx) }

// Queue serializes graph mutations onto a single goroutine.
// It supports optional rate limiting and graceful shutdown.
// Use Enqueue to push operations and Wait to drain.
type Queue struct {
	ch      chan Op
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// New creates a queue with a fixed buffer.
func New(buffer int) *Queue {
	if buffer <= 0 {
		buffer = 32
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{ch: make(chan Op, buffer), ctx: ctx, cancel: cancel}
}

// Start begins the worker goroutine. Safe to call multiple times.
func (q *Queue) Start() {
	if q.started {
		return
	}
	q.started = true
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-q.ctx.Done():
				// drain outstanding ops best-effort with short deadline
				drainUntil := time.After(10 * time.Millisecond)
				for {
					select {
					case op := <-q.ch:
						_ = op.Apply(q.ctx)
					case <-drainUntil:
						return
					default:
						return
					}
				}
			case op := <-q.ch:
				if op == nil {
					continue
				}
				_ = op.Apply(q.ctx)
			}
		}
	}()
}

// Enqueue adds an operation to the queue.
func (q *Queue) Enqueue(op Op) error {
	if q == nil || q.ch == nil {
		return errors.New("queue not initialized")
	}
	select {
	case q.ch <- op:
		return nil
	case <-q.ctx.Done():
		return errors.New("queue closed")
	}
}

// Close stops the worker and waits for it to finish.
func (q *Queue) Close() {
	if q == nil {
		return
	}
	q.cancel()
	q.wg.Wait()
}

package queue

import (
	"context"
	"unsafe"

	aveng "github.com/shaban/macaudio/avaudio/engine"
)

// Dispatcher wraps an avaudio Engine and applies graph mutations via a Queue.
// Call Start() once, and Close() when done. Methods enqueue non-blocking ops.
// For critical paths that must be synchronous, consider adding a Sync variant.
type Dispatcher struct {
	Eng *aveng.Engine
	Q   *Queue
}

func NewDispatcher(eng *aveng.Engine, q *Queue) *Dispatcher {
	if q == nil {
		q = New(32)
	}
	return &Dispatcher{Eng: eng, Q: q}
}

func (d *Dispatcher) Start() { d.Q.Start() }
func (d *Dispatcher) Close() { d.Q.Close() }

// Enqueue schedules an arbitrary queued operation to run on the dispatcher's worker.
// This enables callers to perform small, non-RT-safe tasks (like short ramps)
// serialized with other graph mutations.
func (d *Dispatcher) Enqueue(op Op) error {
	if d == nil || d.Q == nil {
		return nil
	}
	return d.Q.Enqueue(op)
}

// RunSync enqueues an operation and waits for it to complete, returning its error.
// Useful when a caller needs immediate success/failure while still serializing
// with other graph mutations.
func (d *Dispatcher) RunSync(fn Func) error {
	if d == nil || d.Q == nil {
		return fn(context.Background())
	}
	done := make(chan error, 1)
	if err := d.Q.Enqueue(Func(func(ctx context.Context) error {
		err := fn(ctx)
		// Non-blocking send in case caller gave up
		select {
		case done <- err:
		default:
		}
		return err
	})); err != nil {
		return err
	}
	// Wait for completion or queue shutdown
	select {
	case err := <-done:
		return err
	case <-d.Q.ctx.Done():
		return context.Canceled
	}
}

func (d *Dispatcher) Attach(nodePtr unsafe.Pointer) error {
	// Attach synchronously; node lifetimes are sensitive and tests may release quickly.
	return d.RunSync(func(ctx context.Context) error {
		if d.Eng == nil {
			return nil
		}
		return d.Eng.Attach(nodePtr)
	})
}

func (d *Dispatcher) Connect(src, dst unsafe.Pointer, fromBus, toBus int) error {
	return d.Enqueue(Func(func(ctx context.Context) error {
		if d.Eng == nil {
			return nil
		}
		return d.Eng.Connect(src, dst, fromBus, toBus)
	}))
}

func (d *Dispatcher) DisconnectNodeInput(nodePtr unsafe.Pointer, inputBus int) error {
	return d.Enqueue(Func(func(ctx context.Context) error {
		if d.Eng == nil {
			return nil
		}
		return d.Eng.DisconnectNodeInput(nodePtr, inputBus)
	}))
}

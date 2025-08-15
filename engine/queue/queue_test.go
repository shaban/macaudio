package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue_Enqueue_And_Close(t *testing.T) {
	q := New(8)
	q.Start()
	defer q.Close()

	var count int64
	for i := 0; i < 10; i++ {
		if err := q.Enqueue(Func(func(ctx context.Context) error {
			atomic.AddInt64(&count, 1)
			return nil
		})); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	if c := atomic.LoadInt64(&count); c < 10 {
		t.Fatalf("want >=10 ops applied, got %d", c)
	}
}

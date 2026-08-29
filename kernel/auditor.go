package kernel

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// Auditor is an in-memory queue with a background flush. Enqueue must not wait on the adapter.
type Auditor struct {
	store   Store
	ch      chan Envelope
	done    chan struct{}
	once    sync.Once
	dropped atomic.Uint64
}

// NewAuditor starts the background worker. If store is nil, Enqueue is a no-op.
func NewAuditor(store Store, buf int) *Auditor {
	if store == nil {
		return nil
	}
	if buf < 1 {
		buf = 64
	}
	a := &Auditor{
		store: store,
		ch:    make(chan Envelope, buf),
		done:  make(chan struct{}),
	}
	go a.loop()
	return a
}

func (a *Auditor) loop() {
	defer close(a.done)
	for env := range a.ch {
		if err := a.store.Append(context.Background(), env); err != nil {
			log.Printf("audit flush: %v", err)
		}
	}
}

// Enqueue is non-blocking. A full queue drops the event (unflushed buffers are also lost on crash).
func (a *Auditor) Enqueue(env Envelope) {
	if a == nil {
		return
	}
	select {
	case a.ch <- env:
	default:
		n := a.dropped.Add(1)
		log.Printf("audit drop: queue full, dropped=%d", n)
	}
}

// Dropped is the number of envelopes discarded because the queue was full.
func (a *Auditor) Dropped() uint64 {
	if a == nil {
		return 0
	}
	return a.dropped.Load()
}

// Close drains the queue then stops the worker.
func (a *Auditor) Close() {
	if a == nil {
		return
	}
	a.once.Do(func() { close(a.ch) })
	<-a.done
}

// List returns flushed envelopes. The control-plane board uses this; it does not SELECT the storage impl.
func (a *Auditor) List(ctx context.Context, limit int) ([]Envelope, error) {
	if a == nil || a.store == nil {
		return nil, nil
	}
	return a.store.List(ctx, limit)
}

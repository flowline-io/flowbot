// Package bulkhead provides per-capability semaphore isolation for ability invocations.
package bulkhead

import (
	"context"
	"errors"
	"time"
)

// ErrBulkheadFull is returned when the queue is at capacity and a new request cannot be accepted.
var ErrBulkheadFull = errors.New("bulkhead: queue full")

// ErrBulkheadTimeout is returned when a slot cannot be acquired within the configured timeout.
var ErrBulkheadTimeout = errors.New("bulkhead: wait timeout")

type config struct {
	maxConcurrent int
	maxQueue      int
	timeout       time.Duration
	onEnter       func(name string, waitDuration time.Duration)
	onLeave       func(name string)
	onDrop        func(name string, reason string)
	onQueueEnter  func(name string)
	onQueueLeave  func(name string)
}

// Option configures a Bulkhead instance.
type Option func(*config)

// WithMaxConcurrent sets the maximum number of concurrent slots.
func WithMaxConcurrent(n int) Option {
	return func(c *config) { c.maxConcurrent = n }
}

// WithMaxQueue sets the maximum number of requests that can wait for a slot.
func WithMaxQueue(n int) Option {
	return func(c *config) { c.maxQueue = n }
}

// WithTimeout sets the maximum time a request will wait for a slot.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithOnEnter sets a callback invoked when a request acquires a slot.
func WithOnEnter(fn func(name string, waitDuration time.Duration)) Option {
	return func(c *config) { c.onEnter = fn }
}

// WithOnLeave sets a callback invoked when a request releases a slot.
func WithOnLeave(fn func(name string)) Option {
	return func(c *config) { c.onLeave = fn }
}

// WithOnDrop sets a callback invoked when a request is dropped.
func WithOnDrop(fn func(name string, reason string)) Option {
	return func(c *config) { c.onDrop = fn }
}

// WithOnQueueEnter sets a callback invoked when a request enters the wait queue.
func WithOnQueueEnter(fn func(name string)) Option {
	return func(c *config) { c.onQueueEnter = fn }
}

// WithOnQueueLeave sets a callback invoked when a request leaves the wait queue.
func WithOnQueueLeave(fn func(name string)) Option {
	return func(c *config) { c.onQueueLeave = fn }
}

// Bulkhead controls concurrent access for a named resource.
type Bulkhead struct {
	name   string
	sem    chan struct{}
	queue  chan struct{}
	config config
}

// New creates a Bulkhead with the given options.
func New(name string, opts ...Option) *Bulkhead {
	cfg := config{
		maxConcurrent: 1,
		maxQueue:      0,
		timeout:       30 * time.Second,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Bulkhead{
		name:   name,
		sem:    make(chan struct{}, cfg.maxConcurrent),
		queue:  make(chan struct{}, cfg.maxQueue),
		config: cfg,
	}
}

// releaseQueue drains one slot from the wait queue if queueing is enabled.
func (b *Bulkhead) releaseQueue() {
	if b.config.maxQueue > 0 {
		if b.config.onQueueLeave != nil {
			b.config.onQueueLeave(b.name)
		}
		<-b.queue
	}
}

// drop notifies the onDrop callback if configured.
func (b *Bulkhead) drop(reason string) {
	if b.config.onDrop != nil {
		b.config.onDrop(b.name, reason)
	}
}

// Do acquires a slot, waiting up to the configured timeout, then executes fn.
// Returns ErrBulkheadFull if the queue is at capacity.
// Returns ErrBulkheadTimeout if a slot is not acquired within the timeout.
// Returns ctx.Err() if the context is cancelled while waiting.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	if b.config.maxQueue > 0 {
		select {
		case b.queue <- struct{}{}:
			if b.config.onQueueEnter != nil {
				b.config.onQueueEnter(b.name)
			}
		default:
			b.drop("queue_full")
			return ErrBulkheadFull
		}
	}

	waitStart := time.Now()
	timer := time.NewTimer(b.config.timeout)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case b.sem <- struct{}{}:
	case <-timer.C:
		b.releaseQueue()
		if err := ctx.Err(); err != nil {
			b.drop("canceled")
			return err
		}
		b.drop("timeout")
		return ErrBulkheadTimeout
	case <-ctx.Done():
		b.releaseQueue()
		b.drop("canceled")
		return ctx.Err()
	}

	b.releaseQueue()

	defer func() { <-b.sem }()

	if b.config.onEnter != nil {
		b.config.onEnter(b.name, time.Since(waitStart))
	}
	defer func() {
		if b.config.onLeave != nil {
			b.config.onLeave(b.name)
		}
	}()

	return fn()
}

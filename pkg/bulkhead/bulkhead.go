// Package bulkhead provides per-capability semaphore isolation for ability invocations.
package bulkhead

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBulkheadFull    = errors.New("bulkhead: queue full")
	ErrBulkheadTimeout = errors.New("bulkhead: wait timeout")
)

type config struct {
	maxConcurrent int
	maxQueue      int
	timeout       time.Duration
	onEnter       func(name string, waitDuration time.Duration)
	onLeave       func(name string)
	onDrop        func(name string, reason string)
}

type Option func(*config)

func WithMaxConcurrent(n int) Option {
	return func(c *config) { c.maxConcurrent = n }
}

func WithMaxQueue(n int) Option {
	return func(c *config) { c.maxQueue = n }
}

func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

func WithOnEnter(fn func(name string, waitDuration time.Duration)) Option {
	return func(c *config) { c.onEnter = fn }
}

func WithOnLeave(fn func(name string)) Option {
	return func(c *config) { c.onLeave = fn }
}

func WithOnDrop(fn func(name string, reason string)) Option {
	return func(c *config) { c.onDrop = fn }
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

// Do acquires a slot, waiting up to the configured timeout, then executes fn.
// Returns ErrBulkheadFull if the queue is at capacity.
// Returns ErrBulkheadTimeout if a slot is not acquired within the timeout.
// Returns ctx.Err() if the context is cancelled while waiting.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	if b.config.maxQueue > 0 {
		select {
		case b.queue <- struct{}{}:
			defer func() { <-b.queue }()
		default:
			if b.config.onDrop != nil {
				b.config.onDrop(b.name, "queue_full")
			}
			return ErrBulkheadFull
		}
	}

	waitStart := time.Now()
	timer := time.NewTimer(b.config.timeout)
	defer timer.Stop()

	select {
	case b.sem <- struct{}{}:
	case <-timer.C:
		if b.config.onDrop != nil {
			b.config.onDrop(b.name, "timeout")
		}
		return ErrBulkheadTimeout
	case <-ctx.Done():
		if b.config.onDrop != nil {
			b.config.onDrop(b.name, "canceled")
		}
		return ctx.Err()
	}

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

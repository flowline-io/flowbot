package approval

import "sync"

// DefaultDenialThreshold is the consecutive auto-path DENY count that trips the breaker.
const DefaultDenialThreshold = 3

// ReasonBreakerTripped is returned when the denial circuit breaker has latched.
const ReasonBreakerTripped = "approval circuit breaker: too many consecutive denials; stop retrying tools"

// Breaker counts consecutive auto-path denials within one run and latches when tripped.
type Breaker struct {
	mu        sync.Mutex
	threshold int
	count     int
	tripped   bool
}

// NewBreaker creates a run-scoped denial circuit breaker.
func NewBreaker(threshold int) *Breaker {
	if threshold <= 0 {
		threshold = DefaultDenialThreshold
	}
	return &Breaker{threshold: threshold}
}

// Tripped reports whether the breaker has latched for this run.
func (b *Breaker) Tripped() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tripped
}

// Reset clears the consecutive denial count after a successful approve/execute.
// A tripped latch is not cleared within the same run.
func (b *Breaker) Reset() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tripped {
		return
	}
	b.count = 0
}

// RecordDenial increments the consecutive denial count and latches at threshold.
// It returns true when this call trips (or already tripped) the breaker.
func (b *Breaker) RecordDenial() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tripped {
		return true
	}
	b.count++
	if b.count >= b.threshold {
		b.tripped = true
		return true
	}
	return false
}

// Count returns the current consecutive denial count (tests and diagnostics).
func (b *Breaker) Count() int {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

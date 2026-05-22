package pipeline

import (
	"slices"
	"sync"
	"time"
)

// Clock abstracts time operations for testable scheduling.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock delegates to the system clock.
type RealClock struct{}

// NewRealClock returns a new RealClock.
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current system time.
func (*RealClock) Now() time.Time {
	return time.Now()
}

// After returns a channel that fires after duration d.
func (*RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// FakeClock provides deterministic time for tests.
// All timer channels fire in order when Advance is called.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	deadline time.Time
	ch       chan time.Time
}

// NewFakeClock returns a new FakeClock seeded at the given time.
func NewFakeClock(seed time.Time) *FakeClock {
	return &FakeClock{now: seed}
}

// Now returns the current fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// After schedules a timer that fires when Advance is called past its deadline.
func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{
		deadline: c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	c.timers = append(c.timers, t)
	return t.ch
}

// Advance moves the clock forward by d and fires all timers whose deadlines
// are at or before the new time, in chronological order.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	slices.SortFunc(c.timers, func(a, b *fakeTimer) int {
		return a.deadline.Compare(b.deadline)
	})
	var remaining []*fakeTimer
	for _, t := range c.timers {
		if !c.now.Before(t.deadline) {
			t.ch <- t.deadline
		} else {
			remaining = append(remaining, t)
		}
	}
	c.timers = remaining
	c.mu.Unlock()
}

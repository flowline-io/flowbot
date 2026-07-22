package chatagent

import (
	"sync"
)

// ChannelPublisher sends stream events to a buffered channel for SSE writers.
type ChannelPublisher struct {
	mu   sync.Mutex
	ch   chan StreamEvent
	done bool
}

// NewChannelPublisher creates a publisher backed by a buffered event channel.
func NewChannelPublisher(buffer int) *ChannelPublisher {
	if buffer <= 0 {
		buffer = 32
	}
	return &ChannelPublisher{ch: make(chan StreamEvent, buffer)}
}

// Publish enqueues one stream event for the SSE writer goroutine.
// Never blocks: a dead or slow consumer must not stall confirm gates or hub fan-out
// (e.g. after the primary messages SSE disconnects while a run is still waiting).
func (p *ChannelPublisher) Publish(event StreamEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return nil
	}
	select {
	case p.ch <- event:
	default:
		// Drop when the buffer is full. Critical events are still best-effort;
		// observers subscribe on their own channels and stay writable.
	}
	return nil
}

// Events returns the readable event channel.
func (p *ChannelPublisher) Events() <-chan StreamEvent {
	return p.ch
}

// Close marks the publisher done and closes the channel.
func (p *ChannelPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return
	}
	p.done = true
	close(p.ch)
}

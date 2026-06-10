package event

import (
	"context"
	"sync"
)

// Result is the final outcome of an agent run delivered through the stream.
type Result struct {
	Messages []any
	Err      error
}

// Stream multiplexes agent lifecycle events to subscribers and exposes the final result.
type Stream struct {
	events chan Event
	done   chan resultPayload
	subs   []Handler
	mu     sync.Mutex
}

type resultPayload struct {
	Messages []any
	Err      error
}

// NewStream creates a buffered event stream for an agent run.
func NewStream(buffer int) *Stream {
	if buffer <= 0 {
		buffer = 32
	}
	return &Stream{
		events: make(chan Event, buffer),
		done:   make(chan resultPayload, 1),
	}
}

// Events exposes the read-only event channel.
func (s *Stream) Events() <-chan Event {
	return s.events
}

// Subscribe registers a handler invoked sequentially for each emitted event.
func (s *Stream) Subscribe(handler Handler) {
	if handler == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = append(s.subs, handler)
}

// Push emits an event to subscribers and the events channel.
func (s *Stream) Push(ctx context.Context, ev Event) error {
	s.mu.Lock()
	handlers := append([]Handler(nil), s.subs...)
	s.mu.Unlock()

	for _, handler := range handlers {
		if err := handler(ev); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.events <- ev:
		return nil
	}
}

// End closes the stream with the final message list and optional error.
func (s *Stream) End(messages []any, err error) {
	s.done <- resultPayload{Messages: messages, Err: err}
	close(s.events)
}

// Await blocks until the stream ends and returns the final result.
func (s *Stream) Await(ctx context.Context) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case payload := <-s.done:
		return Result{Messages: payload.Messages, Err: payload.Err}, nil
	}
}

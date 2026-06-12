package chatagent

import "context"

// StreamSink receives incremental assistant text during a streaming platform reply.
type StreamSink interface {
	// OnDelta delivers throttled snapshot text for in-progress updates.
	OnDelta(ctx context.Context, text string) error
	// Flush writes the final reply text to the platform message.
	Flush(ctx context.Context, final string) error
}

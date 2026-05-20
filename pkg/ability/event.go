package ability

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

type EventStore interface {
	AppendDataEvent(ctx context.Context, event types.DataEvent) error
	AppendEventOutbox(ctx context.Context, event types.DataEvent) error
}

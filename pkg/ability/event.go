package ability

import "github.com/flowline-io/flowbot/pkg/types"

type EventStore interface {
	AppendDataEvent(event types.DataEvent) error
	AppendEventOutbox(event types.DataEvent) error
}

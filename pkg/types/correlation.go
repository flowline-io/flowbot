package types

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

type lineageKey struct{}

// Lineage is the correlation carried on a context for later data events.
type Lineage struct {
	CorrelationID string
	EventID       string
}

// NewCorrelationID returns a cor_<ULID> identifier.
func NewCorrelationID() string {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "cor_" + Id()
	}
	return "cor_" + id.String()
}

// WithLineage stores a correlation and the current event id on ctx.
func WithLineage(ctx context.Context, correlationID, eventID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, lineageKey{}, Lineage{
		CorrelationID: correlationID,
		EventID:       eventID,
	})
}

// LineageFrom returns the lineage stored on ctx.
func LineageFrom(ctx context.Context) (Lineage, bool) {
	if ctx == nil {
		return Lineage{}, false
	}
	lin, ok := ctx.Value(lineageKey{}).(Lineage)
	if !ok || lin.CorrelationID == "" {
		return Lineage{}, false
	}
	return lin, true
}

// WithTriggerLineage carries the trigger correlation and event id.
// An empty correlation is minted once for the run.
func WithTriggerLineage(ctx context.Context, event DataEvent) context.Context {
	ctx, _ = BindTriggerLineage(ctx, event)
	return ctx
}

// BindTriggerLineage writes a missing correlation onto event and returns ctx for downstream events.
// Downstream events inherit that correlation and use event.EventID as causation.
func BindTriggerLineage(ctx context.Context, event DataEvent) (context.Context, DataEvent) {
	if event.CorrelationID == "" {
		event.CorrelationID = NewCorrelationID()
	}
	return WithLineage(ctx, event.CorrelationID, event.EventID), event
}

// EnsureCorrelation fills a missing correlation from ctx, or mints one.
// An event that already has a correlation is left unchanged.
// A minted id has an empty causation. An inherited id uses the context event id as causation when causation is empty.
func EnsureCorrelation(ctx context.Context, event DataEvent) DataEvent {
	if event.CorrelationID != "" {
		return event
	}
	if lin, ok := LineageFrom(ctx); ok {
		event.CorrelationID = lin.CorrelationID
		if event.CausationID == "" {
			event.CausationID = lin.EventID
		}
		return event
	}
	event.CorrelationID = NewCorrelationID()
	return event
}

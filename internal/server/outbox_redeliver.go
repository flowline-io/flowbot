package server

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	outboxRedeliveryInterval          = 15 * time.Second
	outboxRedeliveryMinAge            = 30 * time.Second
	outboxRedeliveryBatch             = 50
	outboxRedeliveryMaxBatchesPerTick = 40 // up to 2000 events per tick while catching up
)

// outboxStore is the store seam for DataEvent outbox redelivery.
type outboxStore interface {
	ListPendingDataEventOutbox(ctx context.Context, olderThan time.Time, limit int) ([]types.DataEvent, error)
	MarkOutboxPublished(ctx context.Context, eventID string) error
}

// startOutboxRedeliveryLoop periodically republishes unpublished DataEvent outbox rows.
func startOutboxRedeliveryLoop(lc fx.Lifecycle) {
	stop := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go outboxRedeliveryLoop(stop, func() outboxStore {
				if store.Database == nil || store.Database.GetClient() == nil {
					return nil
				}
				return store.EventStoreFromDB()
			}, event.PublishMessage)
			flog.Info("outbox redelivery loop started (interval=%s min_age=%s)", outboxRedeliveryInterval, outboxRedeliveryMinAge)
			return nil
		},
		OnStop: func(_ context.Context) error {
			close(stop)
			return nil
		},
	})
}

func outboxRedeliveryLoop(stop <-chan struct{}, storeFn func() outboxStore, publish dataEventPublisher) {
	ticker := time.NewTicker(outboxRedeliveryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s := storeFn()
			if s == nil {
				continue
			}
			olderThan := time.Now().Add(-outboxRedeliveryMinAge)
			n, more, err := redeliverOutboxCatchUp(context.Background(), s, publish, olderThan, outboxRedeliveryBatch, outboxRedeliveryMaxBatchesPerTick)
			if err != nil {
				flog.Warn("outbox redelivery: %v", err)
				continue
			}
			if n == 0 {
				continue
			}
			if more {
				flog.Info("outbox redelivery: republished %d pending event(s); more remain", n)
			} else {
				flog.Info("outbox redelivery: republished %d pending event(s); backlog cleared", n)
			}
		}
	}
}

// redeliverOutboxCatchUp publishes pending outbox rows in batches until empty or maxBatches.
// more is true when a full final batch suggests unpublished rows may still remain.
func redeliverOutboxCatchUp(
	ctx context.Context,
	s outboxStore,
	publish dataEventPublisher,
	olderThan time.Time,
	batch int,
	maxBatches int,
) (published int, more bool, err error) {
	if batch <= 0 {
		batch = outboxRedeliveryBatch
	}
	if maxBatches <= 0 {
		maxBatches = 1
	}
	for range maxBatches {
		n, err := redeliverPendingOutbox(ctx, s, publish, olderThan, batch)
		if err != nil {
			return published, false, err
		}
		if n == 0 {
			return published, false, nil
		}
		published += n
		if n < batch {
			return published, false, nil
		}
	}
	return published, true, nil
}

// redeliverPendingOutbox publishes unpublished DataEvent outbox rows and marks them published on success.
func redeliverPendingOutbox(
	ctx context.Context,
	s outboxStore,
	publish dataEventPublisher,
	olderThan time.Time,
	limit int,
) (int, error) {
	pending, err := s.ListPendingDataEventOutbox(ctx, olderThan, limit)
	if err != nil {
		return 0, fmt.Errorf("list pending: %w", err)
	}
	published := 0
	for _, de := range pending {
		if err := publish(ctx, DataEventTopic, de); err != nil {
			flog.Error(fmt.Errorf("outbox redelivery: PublishMessage failed event_id=%s: %w", de.EventID, err))
			continue
		}
		if err := s.MarkOutboxPublished(ctx, de.EventID); err != nil {
			flog.Error(fmt.Errorf("outbox redelivery: MarkOutboxPublished failed event_id=%s: %w", de.EventID, err))
			continue
		}
		published++
	}
	return published, nil
}

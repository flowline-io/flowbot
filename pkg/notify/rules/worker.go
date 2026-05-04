package rules

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// Worker periodically scans for expired aggregation windows and flushes them.
type Worker struct {
	engine   *Engine
	interval time.Duration
	onFlush  func(ctx context.Context, ruleID, eventType, channel string, items []map[string]any)
}

// NewWorker creates a new aggregate worker.
func NewWorker(engine *Engine, interval time.Duration, onFlush func(ctx context.Context, ruleID, eventType, channel string, items []map[string]any)) *Worker {
	return &Worker{
		engine:   engine,
		interval: interval,
		onFlush:  onFlush,
	}
}

// Run starts the periodic scan loop. It blocks until the context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scanAndFlush(ctx)
		}
	}
}

func (w *Worker) scanAndFlush(ctx context.Context) {
	keys, err := w.engine.ScanExpiredAggregates(ctx)
	if err != nil {
		flog.Warn("[notify-rules] aggregate scan error: %v", err)
		return
	}

	for _, key := range keys {
		items, err := w.engine.FlushAggregation(ctx, key.RuleID, key.EventType, key.Channel)
		if err != nil {
			flog.Warn("[notify-rules] aggregate flush error: %v", err)
			continue
		}

		if len(items) > 0 && w.onFlush != nil {
			w.onFlush(ctx, key.RuleID, key.EventType, key.Channel, items)
		}
	}
}

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

const DataEventTopic = "pipeline:data_event"

func initPipeline(
	lc fx.Lifecycle,
	cfg *config.Type,
	router *message.Router,
	subscriber message.Subscriber,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
	ac *metrics.AbilityCollector,
	auditor audit.Auditor,
) error {
	pipelineDefs := pipeline.LoadConfig(cfg.Pipelines)
	if len(pipelineDefs) == 0 {
		flog.Info("no pipelines configured, skipping pipeline engine")
		return nil
	}

	var runStore pipeline.RunStore
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			runStore = store.NewPipelineStore(client)
		}
	}

	engine := pipeline.NewEngine(pipelineDefs, runStore, auditor, pc, ec)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			engine.Stop()
			return nil
		},
	})

	ability.SetMetricsCollector(ac)
	ability.SetBulkheadCallbacks()

	poolCfg := cfg.Ability.EventPool
	if err := ability.InitEventPool(poolCfg.Size, poolCfg.ExpiryDuration, ac); err != nil {
		return fmt.Errorf("init event pool: %w", err)
	}

	ability.SetEventEmitter(func(ctx context.Context, result *ability.InvokeResult) {
		if len(result.Events) == 0 {
			return
		}
		desc, ok := hub.Default.Get(result.Capability)
		if !ok {
			return
		}
		for _, ref := range result.Events {
			eventID := ref.EventID
			if eventID == "" {
				eventID = types.Id()
			}

			dataEvent := types.DataEvent{
				EventID:        eventID,
				EventType:      ref.EventType,
				Source:         "ability",
				Capability:     string(result.Capability),
				Operation:      result.Operation,
				Backend:        desc.Backend,
				App:            desc.App,
				EntityID:       ref.EntityID,
				IdempotencyKey: eventID,
				CreatedAt:      time.Now(),
			}

			// Persist to event store
			eventStore := store.NewEventStore(store.Database.GetDB().(*store.Client))
			_ = eventStore.AppendDataEvent(ctx, dataEvent)
			_ = eventStore.AppendEventOutbox(ctx, dataEvent)

			// Publish to Redis Stream via Watermill
			_ = event.PublishMessage(ctx, DataEventTopic, dataEvent)
		}
	})

	// Register pipeline handler in Watermill
	router.AddNoPublisherHandler(
		"onPipelineDataEvent",
		DataEventTopic,
		subscriber,
		func(msg *message.Message) error {
			var dataEvent types.DataEvent
			if err := sonic.Unmarshal(msg.Payload, &dataEvent); err != nil {
				return fmt.Errorf("unmarshal data event: %w", err)
			}

			if ec != nil {
				ec.IncReceived(dataEvent.EventType, dataEvent.Source)
				if !dataEvent.CreatedAt.IsZero() {
					ec.ObserveLag(dataEvent.EventType, time.Since(dataEvent.CreatedAt).Seconds())
				}
			}

			ctx, cancel := context.WithTimeout(msg.Context(), 10*time.Minute)
			defer cancel()
			return engine.Handler()(ctx, dataEvent)
		},
	)

	flog.Info("pipeline engine initialized with %d pipeline(s)", len(pipelineDefs))

	return nil
}

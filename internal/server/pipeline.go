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

	// Register webhook routes.
	if err := registerWebhookRoutes(engine); err != nil {
		return fmt.Errorf("register webhook routes: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
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

	// --- EventSourceManager ---
	srcCollector := metrics.NewEventSourceCollector(nil)

	var srcStateStore *ability.PollingState
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			pollStore := store.NewPollingStateStore(client)
			srcStateStore = ability.NewPollingState(
				&pollingPersistenceAdapter{store: pollStore},
			)
		}
	}
	if srcStateStore == nil {
		srcStateStore = ability.NewPollingState(nil)
	}

	srcMgr := ability.NewEventSourceManager(
		func(ctx context.Context, events []types.DataEvent) error {
			if store.Database == nil || store.Database.GetDB() == nil {
				return nil
			}
			client, ok := store.Database.GetDB().(*store.Client)
			if !ok {
				return nil
			}
			eventStore := store.NewEventStore(client)
			for _, de := range events {
				_ = eventStore.AppendDataEvent(ctx, de)
				_ = eventStore.AppendEventOutbox(ctx, de)
				_ = event.PublishMessage(ctx, DataEventTopic, de)
			}
			return nil
		},
		srcStateStore,
		srcCollector,
	)

	if pool := ability.GetEventPool(); pool != nil {
		srcMgr.SetPool(pool)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return srcMgr.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return srcMgr.Stop(ctx)
		},
	})

	// Register webhook provider route
	sharedApp.Post("/webhook/provider/*", srcMgr.WebhookHandler())
	flog.Info("event source manager initialized")

	return nil
}

// pollingPersistenceAdapter adapts store.PollingStateStore to ability.Persistence.
type pollingPersistenceAdapter struct {
	store *store.PollingStateStore
}

func (a *pollingPersistenceAdapter) LoadAll(ctx context.Context) (map[string]ability.PollingEntry, error) {
	entries, err := a.store.LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]ability.PollingEntry, len(entries))
	for name, e := range entries {
		result[name] = ability.PollingEntry{
			Cursor:      e.Cursor,
			KnownHashes: e.KnownHashes,
		}
	}
	return result, nil
}

func (a *pollingPersistenceAdapter) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	return a.store.Save(ctx, resourceName, cursor, knownHashes)
}

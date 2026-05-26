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
	exampleAdapter "github.com/flowline-io/flowbot/pkg/ability/example/example"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
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
	// Initialize logger before pipeline logic so flog messages are visible at startup.
	_ = initializeLog()

	if err := initEventSourceManager(lc); err != nil {
		return fmt.Errorf("init event source manager: %w", err)
	}

	pipelineDefs := pipeline.LoadConfig(cfg.Pipelines)
	if len(pipelineDefs) == 0 {
		flog.Info("no pipelines configured, skipping pipeline engine")
		return nil
	}

	engine, err := setupPipelineEngine(lc, pipelineDefs, auditor, pc, ec)
	if err != nil {
		return fmt.Errorf("setup pipeline engine: %w", err)
	}

	if err := setupAbilityEmitter(cfg, ac); err != nil {
		return fmt.Errorf("setup ability emitter: %w", err)
	}

	registerPipelineHandler(router, subscriber, engine, ec)

	flog.Info("pipeline engine initialized with %d pipeline(s)", len(pipelineDefs))

	return nil
}

func setupPipelineEngine(
	lc fx.Lifecycle,
	pipelineDefs []pipeline.Definition,
	auditor audit.Auditor,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
) (*pipeline.Engine, error) {
	var runStore pipeline.RunStore
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			runStore = store.NewPipelineStore(client)
		}
	}

	engine := pipeline.NewEngine(pipelineDefs, runStore, auditor, pc, ec)

	if err := registerWebhookRoutes(engine); err != nil {
		return nil, fmt.Errorf("register webhook routes: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			engine.Stop()
			return nil
		},
	})

	return engine, nil
}

func setupAbilityEmitter(cfg *config.Type, ac *metrics.AbilityCollector) error {
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
			eventID := types.Id()

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

			eventStore := store.NewEventStore(store.Database.GetDB().(*store.Client))
			_ = eventStore.AppendDataEvent(ctx, dataEvent)
			_ = eventStore.AppendEventOutbox(ctx, dataEvent)
			_ = event.PublishMessage(ctx, DataEventTopic, dataEvent)
		}
	})

	return nil
}

func registerPipelineHandler(
	router *message.Router,
	subscriber message.Subscriber,
	engine *pipeline.Engine,
	ec *metrics.EventCollector,
) {
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
}

func initEventSourceManager(lc fx.Lifecycle) error {
	srcCollector := metrics.NewEventSourceCollector(nil)

	srcStateStore := buildPollingState()

	srcMgr := ability.NewEventSourceManager(
		func(ctx context.Context, events []types.DataEvent) error {
			if store.Database == nil || store.Database.GetDB() == nil {
				flog.Warn("event_source: emitter skipped, store.Database not ready")
				return nil
			}
			client, ok := store.Database.GetDB().(*store.Client)
			if !ok {
				flog.Warn("event_source: emitter skipped, store.Database is not *store.Client")
				return nil
			}
			eventStore := store.NewEventStore(client)
			for _, de := range events {
				flog.Debug("event_source: storing event %s type=%s source=%s", de.EventID, de.EventType, de.Source)
				if err := eventStore.AppendDataEvent(ctx, de); err != nil {
					flog.Error(fmt.Errorf("event_source: AppendDataEvent failed: %w", err))
				}
				if err := eventStore.AppendEventOutbox(ctx, de); err != nil {
					flog.Error(fmt.Errorf("event_source: AppendEventOutbox failed: %w", err))
				}
				if err := event.PublishMessage(ctx, DataEventTopic, de); err != nil {
					flog.Error(fmt.Errorf("event_source: PublishMessage to %s failed: %w", DataEventTopic, err))
					return fmt.Errorf("event_source: publish failed: %w", err)
				}
			}
			return nil
		},
		srcStateStore,
		srcCollector,
	)

	// Store globally so modules can register webhooks during Bootstrap.
	ability.SetEventSourceManager(srcMgr)

	if pool := ability.GetEventPool(); pool != nil {
		srcMgr.SetPool(pool)
	}

	srcMgr.RegisterWebhook(exampleAdapter.NewExampleWebhook()) // TODO: refactor
	flog.Info("event source: registered example webhook on /webhook/provider/example")

	srcMgr.RegisterPolling(exampleAdapter.NewExamplePoller())
	flog.Info("event source: registered example poller")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return srcMgr.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return srcMgr.Stop(ctx)
		},
	})

	// Register webhook provider route
	sharedAppPtr().Post("/webhook/provider/*", srcMgr.WebhookHandler())
	flog.Info("event source manager initialized")

	return nil
}

func buildPollingState() *ability.PollingState {
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			pollStore := store.NewPollingStateStore(client)
			return ability.NewPollingState(
				&pollingPersistenceAdapter{store: pollStore},
			)
		}
	}
	return ability.NewPollingState(nil)
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

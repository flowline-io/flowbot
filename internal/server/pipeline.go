package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

// pipelineStepCallback publishes pipeline progress events to Redis Streams.
type pipelineStepCallback struct {
	rdb *redis.Client
}

// NewPipelineStepCallback creates a callback backed by the Redis client.
// Returns nil if rdb is nil.
func NewPipelineStepCallback(client *redis.Client) pipeline.StepCallback {
	if client == nil {
		return nil
	}
	return &pipelineStepCallback{rdb: client}
}

func (c *pipelineStepCallback) OnRunStart(_ context.Context, runID int64, pipelineName string,
	_ string, totalSteps int, _ []string) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: -1, Status: "start", TotalSteps: totalSteps,
	}
	c.publish(runID, evt)
	go c.publishExpire(runID, pipeline.StreamTTLFailsafe)
}

func (c *pipelineStepCallback) OnStepStart(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, input map[string]any) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "running", Input: input,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnStepDone(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, output map[string]any, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "done", Output: output, ElapsedMs: elapsedMs,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnStepError(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, err error, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "error", Error: err.Error(), ElapsedMs: elapsedMs,
	}
	c.publish(runID, evt)
}

func (c *pipelineStepCallback) OnRunComplete(_ context.Context, runID int64, pipelineName string,
	elapsedMs int64, failed bool, errMsg string) {
	status := "complete"
	if failed {
		status = "failed"
	}
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: -1, Status: status, ElapsedMs: elapsedMs, Error: errMsg,
	}
	c.publish(runID, evt)
	go c.publishExpire(runID, pipeline.StreamTTLDrain)
}

// publishExpire sets a TTL on the stream. Call with go to avoid blocking the pipeline engine.
func (c *pipelineStepCallback) publishExpire(runID int64, ttl time.Duration) {
	expCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.rdb.Expire(expCtx, pipeline.StreamName(runID), ttl).Err(); err != nil {
		flog.Warn("pipeline live: Expire stream failed run=%d: %v", runID, err)
	}
}

// publish sends a progress event to the per-run Redis Stream asynchronously
// to avoid blocking the pipeline engine on Redis latency or errors.
func (c *pipelineStepCallback) publish(runID int64, evt pipeline.StepProgressEvent) {
	payload, err := sonic.Marshal(evt)
	if err != nil {
		flog.Warn("pipeline live: marshal event failed run=%d step=%s: %v",
			runID, evt.StepName, err)
		return
	}
	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.rdb.XAdd(pubCtx, &redis.XAddArgs{
			Stream: pipeline.StreamName(runID),
			Values: map[string]any{"data": payload},
		}).Err(); err != nil && !errors.Is(err, context.Canceled) {
			flog.Warn("pipeline live: XAdd failed run=%d step=%s status=%s: %v",
				runID, evt.StepName, evt.Status, err)
		}
	}()
}

const DataEventTopic = "pipeline:data_event"

func initPipeline(
	lc fx.Lifecycle,
	cfg *config.Type,
	router *message.Router,
	subscriber message.Subscriber,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
	ac *metrics.CapabilityCollector,
	auditor audit.Auditor,
) error {
	if err := initEventSourceManager(lc); err != nil {
		return fmt.Errorf("init event source manager: %w", err)
	}

	pipelineDefs := loadPipelineDefinitions(context.Background(), cfg)
	engine, err := setupPipelineEngine(lc, pipelineDefs, auditor, pc, ec)
	if err != nil {
		return fmt.Errorf("setup pipeline engine: %w", err)
	}

	pipeline.SetReloadSource(func(ctx context.Context) ([]pipeline.Definition, error) {
		return loadPipelineDefinitions(ctx, cfg), nil
	}, engine)

	if err := setupAbilityEmitter(cfg, ac); err != nil {
		return fmt.Errorf("setup ability emitter: %w", err)
	}

	registerPipelineHandler(router, subscriber, engine, ec)

	flog.Info("pipeline engine initialized with %d pipeline(s)", len(pipelineDefs))

	return nil
}

func loadPipelineDefinitions(ctx context.Context, cfg *config.Type) []pipeline.Definition {
	pipelineDefs := pipeline.LoadConfig(cfg.Pipelines)
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			pipelineDefStore := store.NewPipelineStore(client)
			dbDefs, err := pipeline.LoadFromDB(ctx, pipelineDefStore)
			if err != nil {
				flog.Error(fmt.Errorf("load pipeline defs from db: %w", err))
			} else {
				pipelineDefs = mergeDefinitions(pipelineDefs, dbDefs)
			}
		}
	}
	return pipelineDefs
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

	if rdb.Client != nil {
		engine.SetCallback(NewPipelineStepCallback(rdb.Client))
	}

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

func setupAbilityEmitter(cfg *config.Type, ac *metrics.CapabilityCollector) error {
	capability.SetMetricsCollector(ac)
	capability.SetBulkheadCallbacks()

	poolCfg := cfg.Capability.EventPool
	if err := capability.InitEventPool(poolCfg.Size, poolCfg.ExpiryDuration, ac); err != nil {
		return fmt.Errorf("init event pool: %w", err)
	}

	capability.SetEventEmitter(func(ctx context.Context, result *capability.InvokeResult) {
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
				Source:         "capability",
				Capability:     string(result.Capability),
				Operation:      result.Operation,
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
	router.AddConsumerHandler(
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

	srcMgr := capability.NewEventSourceManager(
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
	capability.SetEventSourceManager(srcMgr)

	if pool := capability.GetEventPool(); pool != nil {
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
	sharedAppPtr().Post("/webhook/provider/*", srcMgr.WebhookHandler())
	flog.Info("event source manager initialized")

	return nil
}

func buildPollingState() *capability.PollingState {
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			pollStore := store.NewPollingStateStore(client)
			return capability.NewPollingState(
				&pollingPersistenceAdapter{store: pollStore},
			)
		}
	}
	return capability.NewPollingState(nil)
}

// pollingPersistenceAdapter adapts store.PollingStateStore to capability.Persistence.
type pollingPersistenceAdapter struct {
	store *store.PollingStateStore
}

func (a *pollingPersistenceAdapter) LoadAll(ctx context.Context) (map[string]capability.PollingEntry, error) {
	entries, err := a.store.LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]capability.PollingEntry, len(entries))
	for name, e := range entries {
		result[name] = capability.PollingEntry{
			Cursor:      e.Cursor,
			KnownHashes: e.KnownHashes,
		}
	}
	return result, nil
}

func (a *pollingPersistenceAdapter) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	return a.store.Save(ctx, resourceName, cursor, knownHashes)
}

func mergeDefinitions(fileDefs, dbDefs []pipeline.Definition) []pipeline.Definition {
	if len(dbDefs) == 0 {
		return fileDefs
	}
	seen := make(map[string]bool, len(dbDefs))
	for _, d := range dbDefs {
		seen[d.Name] = true
	}
	merged := make([]pipeline.Definition, 0, len(fileDefs)+len(dbDefs))
	merged = append(merged, dbDefs...)
	for _, d := range fileDefs {
		if seen[d.Name] {
			continue
		}
		merged = append(merged, d)
		seen[d.Name] = true
	}
	return merged
}

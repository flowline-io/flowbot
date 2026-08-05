package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

const progressPublishTimeout = 2 * time.Second
const progressJobBuffer = 256

type progressJobKind int

const (
	progressJobXAdd progressJobKind = iota
	progressJobExpire
	progressJobSync
)

type progressJob struct {
	kind     progressJobKind
	runID    int64
	payload  []byte
	stepName string
	status   string
	ttl      time.Duration
	done     chan struct{}
}

// pipelineStepCallback publishes pipeline progress events to Redis Streams
// through a single worker so XAdd order matches callback order.
type pipelineStepCallback struct {
	rdb *redis.Client
	ch  chan progressJob
}

// NewPipelineStepCallback creates a callback backed by the Redis client.
// Returns nil if rdb is nil.
func NewPipelineStepCallback(client *redis.Client) pipeline.StepCallback {
	if client == nil {
		return nil
	}
	c := &pipelineStepCallback{
		rdb: client,
		ch:  make(chan progressJob, progressJobBuffer),
	}
	go c.loop()
	return c
}

func (c *pipelineStepCallback) OnRunStart(_ context.Context, runID int64, pipelineName string,
	_ string, totalSteps int, _ []string) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: -1, Status: "start", TotalSteps: totalSteps,
	}
	c.enqueueXAdd(runID, evt)
	c.enqueueExpire(runID, pipeline.StreamTTLFailsafe)
}

func (c *pipelineStepCallback) OnStepStart(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, input map[string]any) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "running", Input: input,
	}
	c.enqueueXAdd(runID, evt)
}

func (c *pipelineStepCallback) OnStepDone(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, output map[string]any, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "done", Output: output, ElapsedMs: elapsedMs,
	}
	c.enqueueXAdd(runID, evt)
}

func (c *pipelineStepCallback) OnStepError(_ context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, err error, elapsedMs int64) {
	evt := pipeline.StepProgressEvent{
		RunID: runID, PipelineName: pipelineName,
		StepIndex: stepIndex, StepName: stepName,
		Status: "error", Error: err.Error(), ElapsedMs: elapsedMs,
	}
	c.enqueueXAdd(runID, evt)
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
	c.enqueueXAdd(runID, evt)
	c.enqueueExpire(runID, pipeline.StreamTTLDrain)
}

func (c *pipelineStepCallback) enqueueXAdd(runID int64, evt pipeline.StepProgressEvent) {
	payload, err := sonic.Marshal(evt)
	if err != nil {
		flog.Warn("pipeline live: marshal event failed run=%d step=%s: %v",
			runID, evt.StepName, err)
		return
	}
	c.ch <- progressJob{
		kind:     progressJobXAdd,
		runID:    runID,
		payload:  payload,
		stepName: evt.StepName,
		status:   evt.Status,
	}
}

func (c *pipelineStepCallback) enqueueExpire(runID int64, ttl time.Duration) {
	c.ch <- progressJob{kind: progressJobExpire, runID: runID, ttl: ttl}
}

// waitIdleForTest drains queued jobs; for tests only.
func (c *pipelineStepCallback) waitIdleForTest() {
	done := make(chan struct{})
	c.ch <- progressJob{kind: progressJobSync, done: done}
	<-done
}

func (c *pipelineStepCallback) loop() {
	for job := range c.ch {
		switch job.kind {
		case progressJobXAdd:
			c.doXAdd(job)
		case progressJobExpire:
			c.doExpire(job)
		case progressJobSync:
			if job.done != nil {
				close(job.done)
			}
		}
	}
}

func (c *pipelineStepCallback) doXAdd(job progressJob) {
	pubCtx, cancel := context.WithTimeout(context.Background(), progressPublishTimeout)
	defer cancel()
	if err := c.rdb.XAdd(pubCtx, &redis.XAddArgs{
		Stream: pipeline.StreamName(job.runID),
		Values: map[string]any{"data": job.payload},
	}).Err(); err != nil && !errors.Is(err, context.Canceled) {
		flog.Warn("pipeline live: XAdd failed run=%d step=%s status=%s: %v",
			job.runID, job.stepName, job.status, err)
	}
}

func (c *pipelineStepCallback) doExpire(job progressJob) {
	expCtx, cancel := context.WithTimeout(context.Background(), progressPublishTimeout)
	defer cancel()
	if err := c.rdb.Expire(expCtx, pipeline.StreamName(job.runID), job.ttl).Err(); err != nil {
		flog.Warn("pipeline live: Expire stream failed run=%d: %v", job.runID, err)
	}
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

	pipelineDefs := loadPipelineDefinitions(context.Background())
	engine, err := setupPipelineEngine(lc, pipelineDefs, auditor, pc, ec)
	if err != nil {
		return fmt.Errorf("setup pipeline engine: %w", err)
	}

	pipeline.SetReloadSource(func(ctx context.Context) ([]pipeline.Definition, error) {
		return loadPipelineDefinitions(ctx), nil
	}, engine)
	pipeline.SetActiveEngine(engine)

	if store.Database != nil && store.Database.GetClient() != nil {
		svc := pipeline.NewService(store.NewPipelineCatalogAdapter(store.PipelineStoreFromDB()))
		pipeline.SetActiveService(svc)
	}

	if err := setupAbilityEmitter(cfg, ac); err != nil {
		return fmt.Errorf("setup ability emitter: %w", err)
	}

	registerPipelineHandler(router, subscriber, engine, ec)
	startOutboxRedeliveryLoop(lc)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			pipeline.SetActiveEngine(nil)
			pipeline.SetActiveService(nil)
			return nil
		},
	})

	flog.Info("pipeline engine initialized with %d pipeline(s)", len(pipelineDefs))

	return nil
}

func loadPipelineDefinitions(ctx context.Context) []pipeline.Definition {
	if store.Database == nil || store.Database.GetClient() == nil {
		return nil
	}
	dbDefs, err := pipeline.LoadFromDB(ctx, store.PipelineStoreFromDB())
	if err != nil {
		flog.Error(fmt.Errorf("load pipeline defs from db: %w", err))
		return nil
	}
	return dbDefs
}

func setupPipelineEngine(
	lc fx.Lifecycle,
	pipelineDefs []pipeline.Definition,
	auditor audit.Auditor,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
) (*pipeline.Engine, error) {
	var runStore pipeline.RunStore
	if store.Database != nil && store.Database.GetClient() != nil {
		runStore = store.NewPipelineRunStoreAdapter(store.PipelineStoreFromDB())
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
		if store.Database == nil || store.Database.GetClient() == nil {
			flog.Warn("capability_emitter: skipped, store.Database not ready")
			return
		}
		eventStore := store.EventStoreFromDB()
		for _, ref := range result.Events {
			eventID := resolveEmittedEventID(ref)

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

			// EventEmitter cannot return errors to the Invoke caller; log and
			// surface publish failure so operators can detect silent drops.
			if err := persistAndPublishDataEvent(ctx, eventStore, event.PublishMessage, DataEventTopic, dataEvent, "capability_emitter"); err != nil {
				flog.Error(err)
			}
		}
	})

	return nil
}

// dataEventPersister is the store seam used by persistAndPublishDataEvent.
type dataEventPersister interface {
	AppendDataEvent(ctx context.Context, de types.DataEvent) error
	AppendEventOutbox(ctx context.Context, de types.DataEvent) error
	MarkOutboxPublished(ctx context.Context, eventID string) error
}

// dataEventPublisher publishes a payload to a topic (typically event.PublishMessage).
type dataEventPublisher func(ctx context.Context, topic string, payload any) error

// persistAndPublishDataEvent writes the audit row + outbox, then publishes to the bus.
// AppendDataEvent / AppendEventOutbox failures abort before publish (no MarkOutboxPublished),
// so redelivery can still pick up durable outbox rows. Publish failure leaves the outbox
// unpublished for the redelivery loop. Mark failure is logged only.
func persistAndPublishDataEvent(
	ctx context.Context,
	persister dataEventPersister,
	publish dataEventPublisher,
	topic string,
	de types.DataEvent,
	logPrefix string,
) error {
	if err := persister.AppendDataEvent(ctx, de); err != nil {
		flog.Error(fmt.Errorf("%s: AppendDataEvent failed event_id=%s: %w", logPrefix, de.EventID, err))
		return fmt.Errorf("%s: AppendDataEvent failed: %w", logPrefix, err)
	}
	if err := persister.AppendEventOutbox(ctx, de); err != nil {
		flog.Error(fmt.Errorf("%s: AppendEventOutbox failed event_id=%s: %w", logPrefix, de.EventID, err))
		return fmt.Errorf("%s: AppendEventOutbox failed: %w", logPrefix, err)
	}
	if err := publish(ctx, topic, de); err != nil {
		flog.Error(fmt.Errorf("%s: PublishMessage to %s failed event_id=%s: %w", logPrefix, topic, de.EventID, err))
		return fmt.Errorf("%s: publish failed: %w", logPrefix, err)
	}
	if err := persister.MarkOutboxPublished(ctx, de.EventID); err != nil {
		flog.Error(fmt.Errorf("%s: MarkOutboxPublished failed event_id=%s: %w", logPrefix, de.EventID, err))
	}
	return nil
}

// resolveEmittedEventID returns EventRef.EventID when set, otherwise a newly generated ID.
func resolveEmittedEventID(ref capability.EventRef) string {
	if id := strings.TrimSpace(ref.EventID); id != "" {
		return id
	}
	return types.Id()
}

// enrichDataEventApp fills DataEvent.App from hub when the converter left it empty.
func enrichDataEventApp(ev *types.DataEvent) {
	if ev == nil || strings.TrimSpace(ev.App) != "" {
		return
	}
	capabilityName := strings.TrimSpace(ev.Capability)
	if capabilityName == "" {
		return
	}
	if desc, ok := hub.Default.Get(hub.CapabilityType(capabilityName)); ok {
		if app := strings.TrimSpace(desc.App); app != "" {
			ev.App = app
			return
		}
	}
	ev.App = capabilityName
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
			if store.Database == nil || store.Database.GetClient() == nil {
				flog.Warn("event_source: emitter skipped, store.Database not ready")
				return nil
			}
			eventStore := store.EventStoreFromDB()
			for i := range events {
				enrichDataEventApp(&events[i])
				de := events[i]
				flog.Debug("event_source: storing event %s type=%s source=%s", de.EventID, de.EventType, de.Source)
				if err := persistAndPublishDataEvent(ctx, eventStore, event.PublishMessage, DataEventTopic, de, "event_source"); err != nil {
					return err
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
	if store.Database != nil && store.Database.GetClient() != nil {
		pollStore := store.NewPollingStateStore(store.Database.GetClient())
		return capability.NewPollingState(
			&pollingPersistenceAdapter{store: pollStore},
		)
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

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flc1125/go-cron/v4"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"

	otelattr "go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

var pooledSonic = sonic.Config{}.Froze()

// CheckpointData is the intermediate state saved at each pipeline step boundary.
type CheckpointData struct {
	StepIndex   int                    `json:"step_index"`
	StepResults map[string]*StepResult `json:"step_results"`
	Event       types.DataEvent        `json:"event"`
	HeartbeatAt time.Time              `json:"heartbeat_at"`
}

// StepResult captures the output of a completed pipeline step.
type StepResult struct {
	Name        string         `json:"name"`
	Capability  string         `json:"capability"`
	Operation   string         `json:"operation"`
	Output      map[string]any `json:"output"`
	CompletedAt time.Time      `json:"completed_at"`
}

// mergeTags merges upstream tags with step-declared tags.
// Upstream tags are the base; step-declared tags override on key collision.
func mergeTags(upstream types.KV, stepTags any) types.KV {
	if upstream == nil {
		upstream = types.KV{}
	}
	stepKV, ok := stepTags.(types.KV)
	if !ok {
		sm, ok := stepTags.(map[string]any)
		if !ok {
			return upstream
		}
		stepKV = types.KV(sm)
	}
	if len(stepKV) == 0 {
		return upstream
	}
	result := make(types.KV, len(upstream)+len(stepKV))
	maps.Copy(result, upstream)
	maps.Copy(result, stepKV)
	return result
}

// RunStore abstracts persistence for pipeline runs, steps, checkpoints and event consumption.
type RunStore interface {
	CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (*gen.PipelineRun, error)
	UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error
	CreateStepRun(ctx context.Context, runID int64, stepName, capName, operation string, params map[string]any, attempt int) (*gen.PipelineStepRun, error)
	UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error
	SaveCheckpoint(ctx context.Context, runID int64, data any) error
	GetIncompleteRuns(ctx context.Context) ([]*gen.PipelineRun, error)
	GetCheckpoint(ctx context.Context, runID int64, target any) error
	GetRun(ctx context.Context, runID int64) (*gen.PipelineRun, error)
	UpdateRunHeartbeat(ctx context.Context, runID int64) error
	HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error)
	RecordConsumption(ctx context.Context, consumerName, eventID string) error
	RecordResourceLink(ctx context.Context, link *gen.ResourceLink) error
}

type Engine struct {
	defs            []Definition
	store           RunStore
	auditor         audit.Auditor
	pipelineMetrics *metrics.PipelineCollector
	eventMetrics    *metrics.EventCollector
	handler         func(ctx context.Context, event types.DataEvent) error
	mu              map[string]*sync.Mutex
	cron            *cron.Cron
	clock           Clock
	callback        StepCallback
	reloadMu        sync.Mutex
}

func NewEngine(defs []Definition, store RunStore, auditor audit.Auditor, pc *metrics.PipelineCollector, ec *metrics.EventCollector) *Engine {
	return NewEngineWithClock(defs, store, auditor, pc, ec, NewRealClock())
}

func NewEngineWithClock(defs []Definition, store RunStore, auditor audit.Auditor, pc *metrics.PipelineCollector, ec *metrics.EventCollector, clock Clock) *Engine {
	e := &Engine{
		defs:            defs,
		store:           store,
		auditor:         auditor,
		pipelineMetrics: pc,
		eventMetrics:    ec,
		mu:              make(map[string]*sync.Mutex),
		clock:           clock,
	}
	e.handler = e.handleEvent

	for _, def := range defs {
		e.mu[def.Name] = &sync.Mutex{}
	}

	e.cron = cron.New(
		cron.WithSeconds(),
		cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)),
	)
	e.registerCronJobs(defs)
	e.cron.Start()
	return e
}

func (e *Engine) registerCronJobs(defs []Definition) {
	for _, def := range defs {
		if !def.Enabled || def.Trigger.Cron == "" {
			continue
		}
		defCopy := def
		_, err := e.cron.AddFunc(def.Trigger.Cron, func(ctx context.Context) error {
			e.executeCronJob(ctx, defCopy)
			return nil
		})
		if err != nil {
			flog.Error(fmt.Errorf("pipeline %s: failed to register cron job %q: %w", def.Name, def.Trigger.Cron, err))
		} else {
			flog.Info("pipeline %s: registered cron trigger %q", def.Name, def.Trigger.Cron)
		}
	}
}

// Reload replaces in-memory pipeline definitions and refreshes cron registrations.
// Event and webhook handlers use the updated defs slice; webhook HTTP routes are not re-registered.
func (e *Engine) Reload(defs []Definition) error {
	if e == nil {
		return nil
	}
	e.reloadMu.Lock()
	defer e.reloadMu.Unlock()

	for _, entry := range e.cron.Entries() {
		e.cron.Remove(entry.ID())
	}

	e.defs = defs
	for _, def := range defs {
		if _, ok := e.mu[def.Name]; !ok {
			e.mu[def.Name] = &sync.Mutex{}
		}
	}
	e.registerCronJobs(defs)
	flog.Info("pipeline engine reloaded with %d definition(s)", len(defs))
	return nil
}

func (e *Engine) Handler() func(ctx context.Context, event types.DataEvent) error {
	return e.handler
}

// SetCallback sets the progress event callback. Pass nil to disable.
func (e *Engine) SetCallback(cb StepCallback) {
	e.callback = cb
}

func (e *Engine) handleEvent(ctx context.Context, event types.DataEvent) error {
	matched := FindByEvent(e.defs, event.EventType)
	if len(matched) == 0 {
		return nil
	}

	for _, def := range matched {
		if e.eventMetrics != nil {
			e.eventMetrics.IncMatched(event.EventType, def.Name)
		}
		func() {
			mu := e.mu[def.Name]
			if mu != nil {
				mu.Lock()
				defer mu.Unlock()
			}
			if err := e.executePipeline(ctx, def, event, "event"); err != nil {
				flog.Error(fmt.Errorf("pipeline %s: %w", def.Name, err))
			}
		}()
	}

	return nil
}

func (e *Engine) executePipeline(ctx context.Context, def Definition, event types.DataEvent, triggerSource string) error {
	ctx, span := trace.StartSpan(ctx, "pipeline."+def.Name+".execute",
		otelattr.String("pipeline.name", def.Name),
		otelattr.String("event.id", event.EventID),
		otelattr.String("event.type", event.EventType),
	)
	defer span.End()

	applyDefinitionUID(def, &event)

	runStart := time.Now()

	e.auditPipelineEvent(ctx, def.Name, "pipeline.start", event.EventID, event.EventType)

	alreadyDone, err := e.checkDedupAndRecord(ctx, def.Name, event.EventID, event.EventType)
	if err != nil {
		return err
	}
	if alreadyDone {
		return nil
	}

	runID, err := e.createRunRecord(ctx, def.Name, event.EventID, event.EventType, triggerSource)
	if err != nil {
		return err
	}

	e.emitRunStart(ctx, runID, &def)

	rc := NewRenderContext(event)
	failed := false
	var finalErr error

	for i, step := range def.Steps {
		e.saveCheckpointIfResumable(ctx, def, event, rc, i, runID)

		if err := e.executeStep(ctx, rc, step, runID, def.Name, i, def.Resumable); err != nil {
			failed = true
			finalErr = err
			break
		}
	}

	if e.pipelineMetrics != nil {
		status := "done"
		if finalErr != nil {
			_, status = terminalStatusFromError(finalErr)
		}
		e.pipelineMetrics.IncRunTotal(def.Name, status)
		e.pipelineMetrics.ObserveRunDuration(def.Name, status, time.Since(runStart).Seconds())
	}

	e.finishRunRecord(ctx, runID, failed, finalErr)

	if e.callback != nil {
		elapsed := time.Since(runStart).Milliseconds()
		var errMsg string
		if finalErr != nil {
			errMsg = finalErr.Error()
		}
		e.callback.OnRunComplete(ctx, runID, def.Name, elapsed, failed, errMsg)
	}

	if finalErr != nil {
		e.auditPipelineEvent(ctx, def.Name, "pipeline.fail", event.EventID, event.EventType)
		return finalErr
	}
	e.auditPipelineEvent(ctx, def.Name, "pipeline.complete", event.EventID, event.EventType)
	return nil
}

func (e *Engine) executeStep(ctx context.Context, rc *RenderContext, step Step, runID int64, pipelineName string, stepIndex int, resumable bool) error {
	ctx, span := trace.StartSpan(ctx, "pipeline."+pipelineName+".step."+step.Name,
		otelattr.String("pipeline.step.name", step.Name),
		otelattr.String("pipeline.step.capability", string(step.Capability)),
		otelattr.String("pipeline.step.operation", step.Operation),
	)
	defer span.End()

	stepStart := time.Now()

	renderedParams, err := rc.RenderParams(step.Params)
	if err != nil {
		return fmt.Errorf("render params step %s: %w", step.Name, err)
	}
	InjectAgentRunDefaults(step, renderedParams, rc, pipelineName)

	if e.callback != nil {
		e.callback.OnStepStart(ctx, runID, pipelineName, stepIndex, step.Name, renderedParams)
	}

	if capability.IsMutation(step.Operation) && len(rc.Event.Tags) > 0 {
		injectTags(rc, renderedParams)
	}

	attempt := 1
	stepRunID, err := e.createStepRunRecord(ctx, runID, step.Name, string(step.Capability), step.Operation, renderedParams, attempt)
	if err != nil {
		return err
	}

	// Start heartbeat goroutine for long-running steps.
	var hbCtx context.Context
	var hbCancel context.CancelFunc
	if resumable && e.store != nil && runID != 0 {
		hbCtx, hbCancel = context.WithCancel(ctx)
		defer hbCancel()
		go e.heartbeatLoop(hbCtx, runID, pipelineName)
	}

	if e.pipelineMetrics != nil {
		e.pipelineMetrics.IncStepTotal(pipelineName, step.Name, "start")
	}

	retryCfg := step.Retry
	if retryCfg == nil {
		retryCfg = &backoff.Config{MaxAttempts: 0}
	}
	boCfg := *retryCfg
	boCfg.OnRetry = func(a int, d time.Duration, err error) {
		flog.Info("pipeline %s step %s attempt %d failed, retrying in %v: %v",
			pipelineName, step.Name, a, d, err)
	}

	var stepResult map[string]any
	var stepResource *capability.ResourceMeta
	attempt, retryErr := backoff.Do(ctx, boCfg, func(ctx context.Context) error {
		res, invokeErr := capability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
		if invokeErr != nil {
			trace.RecordError(ctx, invokeErr)
			return invokeErr
		}
		stepResult = extractResult(res)
		stepResource = res.Resource
		return nil
	})

	if retryErr != nil {
		stepErr := formatStepError(step.Name, retryErr, attempt)
		e.recordStepFailure(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), retryErr, attempt, stepStart)
		if e.callback != nil {
			e.callback.OnStepError(ctx, runID, pipelineName, stepIndex, step.Name, stepErr, time.Since(stepStart).Milliseconds())
		}
		return stepErr
	}

	rc.RecordStepResult(step.Name, stepResult)
	e.saveResourceLink(ctx, rc, step, stepResource, runID, pipelineName)
	e.recordStepSuccess(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), stepResult, attempt, stepStart)
	if e.callback != nil {
		e.callback.OnStepDone(ctx, runID, pipelineName, stepIndex, step.Name, stepResult, time.Since(stepStart).Milliseconds())
	}
	flog.Info("pipeline %s step %s completed (attempt %d)", pipelineName, step.Name, attempt)
	return nil
}

// injectTags merges event tags into rendered params for mutation steps.
func injectTags(rc *RenderContext, renderedParams map[string]any) {
	renderedParams["tags"] = mergeTags(rc.Event.Tags, renderedParams["tags"])
}

// InjectAgentRunDefaults applies default uid and memory_scope for agent.run and notify.send steps.
func InjectAgentRunDefaults(step Step, renderedParams map[string]any, rc *RenderContext, pipelineName string) {
	injectAgentRunMemoryScope(step, renderedParams, pipelineName)
	injectEventUID(step, renderedParams, rc)
}

// applyDefinitionUID sets event.UID from the pipeline definition when the trigger event has no UID.
func applyDefinitionUID(def Definition, event *types.DataEvent) {
	if event == nil {
		return
	}
	if strings.TrimSpace(event.UID) != "" {
		return
	}
	if uid := strings.TrimSpace(def.UID); uid != "" {
		event.UID = uid
	}
}

func injectAgentRunMemoryScope(step Step, renderedParams map[string]any, pipelineName string) {
	if step.Capability != hub.CapAgent || step.Operation != capability.OpAgentRun {
		return
	}
	if renderedParams == nil {
		return
	}
	if raw, ok := renderedParams["memory_scope"]; ok && raw != nil {
		if scope, ok := raw.(string); ok && strings.TrimSpace(scope) != "" {
			return
		}
	}
	renderedParams["memory_scope"] = pipelineName
}

// injectEventUID copies Event.UID into step params for agent.run and notify.send when unset.
func injectEventUID(step Step, renderedParams map[string]any, rc *RenderContext) {
	if !stepNeedsEventUID(step) {
		return
	}
	if renderedParams == nil || rc == nil {
		return
	}
	if raw, ok := renderedParams["uid"]; ok && raw != nil {
		if uid, ok := raw.(string); ok && strings.TrimSpace(uid) != "" {
			return
		}
	}
	uid := strings.TrimSpace(rc.Event.UID)
	if uid == "" {
		return
	}
	renderedParams["uid"] = uid
}

// stepNeedsEventUID reports whether the step should receive a default uid from the event.
func stepNeedsEventUID(step Step) bool {
	switch {
	case step.Capability == hub.CapAgent && step.Operation == capability.OpAgentRun:
		return true
	case step.Capability == hub.CapNotify && step.Operation == "send":
		return true
	default:
		return false
	}
}

// saveResourceLink records a resource link when a capability step reports a created resource.
func (e *Engine) saveResourceLink(ctx context.Context, rc *RenderContext, step Step, stepResource *capability.ResourceMeta, runID int64, pipelineName string) {
	if stepResource == nil || stepResource.EntityID == "" || stepResource.EventID == "" {
		return
	}
	if e.store == nil {
		return
	}
	link := &gen.ResourceLink{
		SourceEventID:    rc.Event.EventID,
		TargetEventID:    stepResource.EventID,
		SourceApp:        rc.Event.App,
		TargetApp:        stepResource.App,
		SourceCapability: rc.Event.Capability,
		TargetCapability: string(step.Capability),
		SourceEntityID:   rc.Event.EntityID,
		TargetEntityID:   stepResource.EntityID,
		PipelineRunID:    runID,
		PipelineName:     pipelineName,
	}
	if err := e.store.RecordResourceLink(ctx, link); err != nil {
		flog.Error(fmt.Errorf("record resource link pipeline %s: %w", pipelineName, err))
	}
}

// formatStepError builds a descriptive error message from a step invoke failure.
func formatStepError(stepName string, err error, attempt int) error {
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("step %s cancelled: %w", stepName, err)
	case attempt > 1:
		return fmt.Errorf("step %s (retries exhausted): %w", stepName, err)
	default:
		return fmt.Errorf("step %s: %w", stepName, err)
	}
}

func (e *Engine) heartbeatLoop(ctx context.Context, runID int64, pipelineName string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.store.UpdateRunHeartbeat(ctx, runID); err != nil {
				flog.Error(fmt.Errorf("heartbeat pipeline %s run %d: %w", pipelineName, runID, err))
			}
		}
	}
}

func buildStepResults(rc *RenderContext) map[string]*StepResult {
	result := make(map[string]*StepResult, len(rc.Steps))
	for name, data := range rc.Steps {
		result[name] = &StepResult{
			Name:   name,
			Output: data,
		}
	}
	return result
}

func (e *Engine) createRunRecord(ctx context.Context, name, eventID, eventType, triggerSource string) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	run, err := e.store.CreateRun(ctx, name, eventID, eventType, triggerSource)
	if err != nil {
		return 0, fmt.Errorf("create run: %w", err)
	}
	return run.ID, nil
}

func (e *Engine) createStepRunRecord(ctx context.Context, runID int64, stepName, capName, operation string, params map[string]any, attempt int) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	paramsJSON := convertToTypesKV(params)
	sr, err := e.store.CreateStepRun(ctx, runID, stepName, capName, operation, paramsJSON, attempt)
	if err != nil {
		return 0, fmt.Errorf("create step run %s: %w", stepName, err)
	}
	return sr.ID, nil
}

func (e *Engine) updateStepRunRecord(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) {
	if e.store == nil || stepRunID == 0 {
		return
	}
	var resultJSON map[string]any
	if result != nil {
		resultJSON = convertToTypesKV(result)
	}
	if err := e.store.UpdateStepRun(ctx, stepRunID, status, resultJSON, errMsg, attempt); err != nil {
		flog.Error(fmt.Errorf("update step run %d: %w", stepRunID, err))
	}
}

func (e *Engine) finishRunRecord(ctx context.Context, runID int64, failed bool, finalErr error) {
	if e.store == nil || runID == 0 {
		return
	}
	status := int(schema.PipelineDone)
	errMsg := ""
	if failed {
		st, _ := terminalStatusFromError(finalErr)
		status = int(st)
		if finalErr != nil {
			errMsg = finalErr.Error()
		}
	}
	if err := e.store.UpdateRunStatus(ctx, runID, status, errMsg); err != nil {
		flog.Error(fmt.Errorf("update run %d status: %w", runID, err))
	}
}

// terminalStatusFromError maps a terminal run/step error to persistence status and metrics label.
// Context cancellation and deadline exceeded are recorded as cancelled; all other errors as failed.
func terminalStatusFromError(err error) (schema.PipelineState, string) {
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return schema.PipelineCancel, "cancel"
	}
	return schema.PipelineFailed, "failed"
}

func (e *Engine) auditPipelineEvent(ctx context.Context, pipelineName, action, eventID, eventType string) {
	if e.auditor == nil {
		return
	}
	_ = e.auditor.Record(ctx, audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "pipeline",
			SubjectID:   "system:pipeline",
		},
		Action: action,
		Target: audit.Target{Type: "pipeline", ID: pipelineName},
		Request: map[string]any{
			"event_id":   eventID,
			"event_type": eventType,
		},
	})
}

func (e *Engine) checkDedupAndRecord(ctx context.Context, pipelineName, eventID, eventType string) (bool, error) {
	if e.store == nil {
		return false, nil
	}
	consumed, err := e.store.HasConsumed(ctx, pipelineName, eventID)
	if err != nil {
		return false, fmt.Errorf("check consumption: %w", err)
	}
	if consumed {
		flog.Info("pipeline %s already consumed event %s", pipelineName, eventID)
		if e.eventMetrics != nil {
			e.eventMetrics.IncDedup(eventType, pipelineName)
		}
		return true, nil
	}
	if err := e.store.RecordConsumption(ctx, pipelineName, eventID); err != nil {
		return false, fmt.Errorf("record consumption: %w", err)
	}
	return false, nil
}

func (e *Engine) saveCheckpointIfResumable(ctx context.Context, def Definition, event types.DataEvent, rc *RenderContext, stepIndex int, runID int64) {
	if !def.Resumable || e.store == nil || runID == 0 {
		return
	}
	cp := &CheckpointData{
		StepIndex:   stepIndex,
		StepResults: buildStepResults(rc),
		Event:       event,
		HeartbeatAt: e.clock.Now(),
	}
	if cpErr := e.store.SaveCheckpoint(ctx, runID, cp); cpErr != nil {
		flog.Error(fmt.Errorf("save checkpoint pipeline %s step %d: %w", def.Name, stepIndex, cpErr))
	}
}

func (e *Engine) recordStepMetrics(pipelineName, stepName, capName, status string, durationSec float64, attempt int) {
	if e.pipelineMetrics == nil {
		return
	}
	e.pipelineMetrics.IncStepTotal(pipelineName, stepName, status)
	e.pipelineMetrics.ObserveStepDuration(pipelineName, stepName, capName, status, durationSec)
	if attempt > 1 {
		e.pipelineMetrics.IncStepRetry(pipelineName, stepName)
	}
}

func (e *Engine) recordStepSuccess(ctx context.Context, stepRunID int64, pipelineName, stepName, capName string, stepResult map[string]any, attempt int, stepStart time.Time) {
	e.updateStepRunRecord(ctx, stepRunID, int(schema.PipelineDone), stepResult, "", attempt)
	e.recordStepMetrics(pipelineName, stepName, capName, "done", time.Since(stepStart).Seconds(), attempt)
}

func (e *Engine) recordStepFailure(ctx context.Context, stepRunID int64, pipelineName, stepName, capName string, err error, attempt int, stepStart time.Time) {
	status, metricStatus := terminalStatusFromError(err)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	e.updateStepRunRecord(ctx, stepRunID, int(status), nil, errMsg, attempt)
	e.recordStepMetrics(pipelineName, stepName, capName, metricStatus, time.Since(stepStart).Seconds(), attempt)
}

func extractResult(res *capability.InvokeResult) map[string]any {
	return StepResultFromInvoke(res)
}

func convertToTypesKV(m map[string]any) types.KV {
	result := make(types.KV, len(m))
	maps.Copy(result, m)
	return result
}

// ResumePipeline attempts to resume a pipeline run from its last checkpoint.
// It reloads the checkpoint, reconstructs the RenderContext, and continues
// from the checkpointed step index.
func (e *Engine) ResumePipeline(ctx context.Context, runID int64) error {
	if e.store == nil {
		return fmt.Errorf("pipeline store not available")
	}

	run, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run %d: %w", runID, err)
	}

	if err := e.store.UpdateRunStatus(ctx, runID, int(schema.PipelineStart), ""); err != nil {
		flog.Error(fmt.Errorf("update run %d status to started for resume: %w", runID, err))
	}

	if e.pipelineMetrics != nil {
		e.pipelineMetrics.IncResume(run.PipelineName)
	}

	cp := &CheckpointData{}
	if err := e.store.GetCheckpoint(ctx, runID, cp); err != nil {
		return fmt.Errorf("load checkpoint for run %d: %w", runID, err)
	}
	if cp.StepIndex < 0 {
		return fmt.Errorf("invalid checkpoint for run %d", runID)
	}

	def := e.findResumableDef(run.PipelineName)
	if def == nil {
		return fmt.Errorf("no resumable pipeline definition for %s (run %d)", run.PipelineName, runID)
	}

	mu := e.mu[def.Name]
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	rc := NewRenderContext(cp.Event)
	for name, sr := range cp.StepResults {
		rc.RecordStepResult(name, sr.Output)
	}

	e.emitRunStart(ctx, runID, def)

	startTime := e.clock.Now()
	failed, finalErr := e.runResumeSteps(ctx, rc, def, runID, cp)

	e.finishRunRecord(ctx, runID, failed, finalErr)
	e.emitRunComplete(ctx, runID, def, startTime, failed, finalErr)

	return finalErr
}

// findResumableDef returns the first resumable pipeline definition matching the given name.
func (e *Engine) findResumableDef(name string) *Definition {
	for i := range e.defs {
		if e.defs[i].Resumable && e.defs[i].Name == name {
			return &e.defs[i]
		}
	}
	return nil
}

// runResumeSteps executes pipeline steps from the checkpoint index, persisting
// checkpoint state before each step. Returns whether the run failed and any error.
func (e *Engine) runResumeSteps(ctx context.Context, rc *RenderContext, def *Definition, runID int64, cp *CheckpointData) (bool, error) {
	for i := cp.StepIndex; i < len(def.Steps); i++ {
		step := def.Steps[i]
		if cpErr := e.store.SaveCheckpoint(ctx, runID, &CheckpointData{
			StepIndex:   i,
			StepResults: buildStepResults(rc),
			Event:       cp.Event,
			HeartbeatAt: e.clock.Now(),
		}); cpErr != nil {
			flog.Error(fmt.Errorf("save checkpoint during resume run %d step %d: %w", runID, i, cpErr))
		}

		if err := e.executeStep(ctx, rc, step, runID, def.Name, i, true); err != nil {
			return true, err
		}
	}
	return false, nil
}

// RegisterWebhooks returns a map of webhook path to pipeline Definition for
// all webhook-enabled pipelines. Duplicate paths return an error.
func (e *Engine) RegisterWebhooks() (map[string]*Definition, error) {
	result := make(map[string]*Definition)
	for i := range e.defs {
		if e.defs[i].Trigger.Webhook == nil {
			continue
		}
		path := e.defs[i].Trigger.Webhook.Path
		if _, exists := result[path]; exists {
			return nil, fmt.Errorf("duplicate webhook path %q", path)
		}
		result[path] = &e.defs[i]
	}
	return result, nil
}

// Stop shuts down the cron scheduler. It waits up to 30 seconds for
// in-flight jobs to complete, then force-cancels and logs a warning.
func (e *Engine) Stop() {
	if e.cron == nil {
		return
	}
	ctx := e.cron.Stop()
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
		flog.Warn("pipeline cron stop timed out after 30s, forcing shutdown")
	}
}

func (e *Engine) executeCronJob(_ context.Context, def Definition) {
	mu := e.mu[def.Name]
	if !mu.TryLock() {
		if e.pipelineMetrics != nil {
			e.pipelineMetrics.IncCronSkip(def.Name)
		}
		flog.Info("pipeline %s: cron tick skipped, previous run still in progress", def.Name)
		return
	}
	defer mu.Unlock()

	start := e.clock.Now()

	dataEvent := types.DataEvent{
		EventID:   types.Id(),
		EventType: fmt.Sprintf("pipeline.cron:%s", def.Name),
		Source:    "cron",
		CreatedAt: e.clock.Now(),
	}

	timeout := def.Trigger.CronTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	execCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := e.executePipeline(execCtx, def, dataEvent, "cron")
	if e.pipelineMetrics != nil {
		status := "done"
		if err != nil {
			_, status = terminalStatusFromError(err)
		}
		e.pipelineMetrics.IncCronExec(def.Name, status)
		e.pipelineMetrics.ObserveCronDuration(def.Name, e.clock.Now().Sub(start).Seconds())
	}
	if err != nil {
		flog.Error(fmt.Errorf("pipeline %s cron run: %w", def.Name, err))
	}
}

// ExecuteWebhook executes a pipeline from a webhook trigger. It uses the
// per-pipeline mutex for concurrency control and calls executePipeline
// with a synthetic event.
func (e *Engine) ExecuteWebhook(ctx context.Context, def *Definition, event types.DataEvent) error {
	mu := e.mu[def.Name]
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	return e.executePipeline(ctx, *def, event, "webhook")
}

// triggerDescription returns a human-readable trigger description string.
func triggerDescription(t Trigger) string {
	if t.Event != "" {
		return "event:" + t.Event
	}
	if t.Webhook != nil && t.Webhook.Path != "" {
		return "webhook:" + t.Webhook.Path
	}
	if t.Cron != "" {
		return "cron:" + t.Cron
	}
	return "unknown"
}

// emitRunStart emits a run-level start event via the callback.
func (e *Engine) emitRunStart(ctx context.Context, runID int64, def *Definition) {
	if e.callback == nil {
		return
	}
	triggerDesc := triggerDescription(def.Trigger)
	stepNames := make([]string, len(def.Steps))
	for i, s := range def.Steps {
		stepNames[i] = s.Name
	}
	e.callback.OnRunStart(ctx, runID, def.Name, triggerDesc, len(def.Steps), stepNames)
}

// emitRunComplete emits a run-level completion event via the callback.
func (e *Engine) emitRunComplete(ctx context.Context, runID int64, def *Definition,
	startTime time.Time, failed bool, finalErr error) {
	if e.callback == nil {
		return
	}
	elapsed := e.clock.Now().Sub(startTime).Milliseconds()
	var errMsg string
	if finalErr != nil {
		errMsg = finalErr.Error()
	}
	e.callback.OnRunComplete(ctx, runID, def.Name, elapsed, failed, errMsg)
}

// MutexFor returns the per-pipeline mutex for the given pipeline name.
// Exported for testing (BDD specs).
func (e *Engine) MutexFor(name string) *sync.Mutex {
	return e.mu[name]
}

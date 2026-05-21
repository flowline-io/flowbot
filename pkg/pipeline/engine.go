package pipeline

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"

	otelattr "go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/internal/store/model"
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

// RunStore abstracts persistence for pipeline runs, steps, checkpoints and event consumption.
type RunStore interface {
	CreateRun(ctx context.Context, pipelineName, eventID, eventType string) (*model.PipelineRun, error)
	UpdateRunStatus(ctx context.Context, runID int64, status model.PipelineState, errMsg string) error
	CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error)
	UpdateStepRun(ctx context.Context, stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(ctx context.Context, runID int64, data any) error
	GetIncompleteRuns(ctx context.Context) ([]*model.PipelineRun, error)
	GetCheckpoint(ctx context.Context, runID int64, target any) error
	GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error)
	UpdateRunHeartbeat(ctx context.Context, runID int64) error
	HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error)
	RecordConsumption(ctx context.Context, consumerName, eventID string) error
}

type Engine struct {
	defs            []Definition
	store           RunStore
	auditor         audit.Auditor
	pipelineMetrics *metrics.PipelineCollector
	eventMetrics    *metrics.EventCollector
	handler         func(ctx context.Context, event types.DataEvent) error
}

func NewEngine(defs []Definition, store RunStore, auditor audit.Auditor, pc *metrics.PipelineCollector, ec *metrics.EventCollector) *Engine {
	e := &Engine{
		defs:            defs,
		store:           store,
		auditor:         auditor,
		pipelineMetrics: pc,
		eventMetrics:    ec,
	}
	e.handler = e.handleEvent
	return e
}

func (e *Engine) Handler() func(ctx context.Context, event types.DataEvent) error {
	return e.handler
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
		if err := e.executePipeline(ctx, def, event); err != nil {
			flog.Error(fmt.Errorf("pipeline %s: %w", def.Name, err))
		}
	}

	return nil
}

func (e *Engine) executePipeline(ctx context.Context, def Definition, event types.DataEvent) error {
	ctx, span := trace.StartSpan(ctx, "pipeline."+def.Name+".execute",
		otelattr.String("pipeline.name", def.Name),
		otelattr.String("event.id", event.EventID),
		otelattr.String("event.type", event.EventType),
	)
	defer span.End()

	runStart := time.Now()

	e.auditPipelineEvent(ctx, def.Name, "pipeline.start", event.EventID, event.EventType)

	alreadyDone, err := e.checkDedupAndRecord(ctx, def.Name, event.EventID, event.EventType)
	if err != nil {
		return err
	}
	if alreadyDone {
		return nil
	}

	runID, err := e.createRunRecord(ctx, def.Name, event.EventID, event.EventType)
	if err != nil {
		return err
	}

	rc := NewRenderContext(event)
	failed := false
	var finalErr error

	for i, step := range def.Steps {
		e.saveCheckpointIfResumable(ctx, def, event, rc, i, runID)

		if err := e.executeStep(ctx, rc, step, runID, def.Name, def.Resumable); err != nil {
			failed = true
			finalErr = err
			break
		}
	}

	if e.pipelineMetrics != nil {
		status := "done"
		if finalErr != nil {
			status = "cancel"
		}
		e.pipelineMetrics.IncRunTotal(def.Name, status)
		e.pipelineMetrics.ObserveRunDuration(def.Name, status, time.Since(runStart).Seconds())
	}

	e.finishRunRecord(ctx, runID, failed, finalErr)

	if finalErr != nil {
		e.auditPipelineEvent(ctx, def.Name, "pipeline.fail", event.EventID, event.EventType)
		return finalErr
	}
	e.auditPipelineEvent(ctx, def.Name, "pipeline.complete", event.EventID, event.EventType)
	return nil
}

func (e *Engine) executeStep(ctx context.Context, rc *RenderContext, step Step, runID int64, pipelineName string, resumable bool) error {
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
	attempt, retryErr := backoff.Do(ctx, boCfg, func(ctx context.Context) error {
		res, invokeErr := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
		if invokeErr != nil {
			trace.RecordError(ctx, invokeErr)
			return invokeErr
		}
		stepResult = extractResult(res)
		return nil
	})

	if retryErr != nil {
		var stepErr error
		switch {
		case errors.Is(retryErr, context.Canceled) || errors.Is(retryErr, context.DeadlineExceeded):
			stepErr = fmt.Errorf("step %s cancelled: %w", step.Name, retryErr)
		case attempt > 1:
			stepErr = fmt.Errorf("step %s (retries exhausted): %w", step.Name, retryErr)
		default:
			stepErr = fmt.Errorf("step %s: %w", step.Name, retryErr)
		}
		e.recordStepFailure(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), retryErr.Error(), attempt, stepStart)
		return stepErr
	}

	rc.RecordStepResult(step.Name, stepResult)
	e.recordStepSuccess(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), stepResult, attempt, stepStart)
	flog.Info("pipeline %s step %s completed (attempt %d)", pipelineName, step.Name, attempt)
	return nil
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

func (e *Engine) createRunRecord(ctx context.Context, name, eventID, eventType string) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	run, err := e.store.CreateRun(ctx, name, eventID, eventType)
	if err != nil {
		return 0, fmt.Errorf("create run: %w", err)
	}
	return run.ID, nil
}

func (e *Engine) createStepRunRecord(ctx context.Context, runID int64, stepName, capability, operation string, params map[string]any, attempt int) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	paramsJSON := model.JSON(convertToTypesKV(params))
	sr, err := e.store.CreateStepRun(ctx, runID, stepName, capability, operation, paramsJSON, attempt)
	if err != nil {
		return 0, fmt.Errorf("create step run %s: %w", stepName, err)
	}
	return sr.ID, nil
}

func (e *Engine) updateStepRunRecord(ctx context.Context, stepRunID int64, status model.PipelineState, result map[string]any, errMsg string, attempt int) {
	if e.store == nil || stepRunID == 0 {
		return
	}
	var resultJSON model.JSON
	if result != nil {
		resultJSON = model.JSON(convertToTypesKV(result))
	}
	_ = e.store.UpdateStepRun(ctx, stepRunID, status, resultJSON, errMsg, attempt)
}

func (e *Engine) finishRunRecord(ctx context.Context, runID int64, failed bool, finalErr error) {
	if e.store == nil || runID == 0 {
		return
	}
	status := model.PipelineDone
	errMsg := ""
	if failed {
		status = model.PipelineCancel
		if finalErr != nil {
			errMsg = finalErr.Error()
		}
	}
	_ = e.store.UpdateRunStatus(ctx, runID, status, errMsg)
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
		HeartbeatAt: time.Now(),
	}
	if cpErr := e.store.SaveCheckpoint(ctx, runID, cp); cpErr != nil {
		flog.Error(fmt.Errorf("save checkpoint pipeline %s step %d: %w", def.Name, stepIndex, cpErr))
	}
}

func (e *Engine) recordStepMetrics(pipelineName, stepName, capability, status string, durationSec float64, attempt int) {
	if e.pipelineMetrics == nil {
		return
	}
	e.pipelineMetrics.IncStepTotal(pipelineName, stepName, status)
	e.pipelineMetrics.ObserveStepDuration(pipelineName, stepName, capability, status, durationSec)
	if attempt > 1 {
		e.pipelineMetrics.IncStepRetry(pipelineName, stepName)
	}
}

func (e *Engine) recordStepSuccess(ctx context.Context, stepRunID int64, pipelineName, stepName, capability string, stepResult map[string]any, attempt int, stepStart time.Time) {
	e.updateStepRunRecord(ctx, stepRunID, model.PipelineDone, stepResult, "", attempt)
	e.recordStepMetrics(pipelineName, stepName, capability, "done", time.Since(stepStart).Seconds(), attempt)
}

func (e *Engine) recordStepFailure(ctx context.Context, stepRunID int64, pipelineName, stepName, capability, errMsg string, attempt int, stepStart time.Time) {
	e.updateStepRunRecord(ctx, stepRunID, model.PipelineCancel, nil, errMsg, attempt)
	e.recordStepMetrics(pipelineName, stepName, capability, "cancel", time.Since(stepStart).Seconds(), attempt)
}

func extractResult(res *ability.InvokeResult) map[string]any {
	if res.Data == nil {
		return map[string]any{}
	}
	if m, ok := res.Data.(map[string]any); ok {
		return m
	}
	// For non-map types (e.g. slices, structs), serialize via JSON and store
	// in a map so template resolution can access individual fields.
	dataJSON, err := pooledSonic.Marshal(res.Data)
	if err != nil {
		return map[string]any{"result": res.Data}
	}
	var stepResult any
	if err := pooledSonic.Unmarshal(dataJSON, &stepResult); err != nil {
		return map[string]any{"result": res.Data}
	}
	if m, ok := stepResult.(map[string]any); ok {
		return m
	}
	return map[string]any{"items": stepResult}
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

	// Find the pipeline definition matching this run's pipeline name.
	var def *Definition
	for i := range e.defs {
		if e.defs[i].Resumable && e.defs[i].Name == run.PipelineName {
			def = &e.defs[i]
			break
		}
	}
	if def == nil {
		return fmt.Errorf("no resumable pipeline definition for %s (run %d)", run.PipelineName, runID)
	}

	rc := NewRenderContext(cp.Event)
	for name, sr := range cp.StepResults {
		rc.RecordStepResult(name, sr.Output)
	}

	failed := false
	var finalErr error
	for i := cp.StepIndex; i < len(def.Steps); i++ {
		step := def.Steps[i]
		if cpErr := e.store.SaveCheckpoint(ctx, runID, &CheckpointData{
			StepIndex:   i,
			StepResults: buildStepResults(rc),
			Event:       cp.Event,
			HeartbeatAt: time.Now(),
		}); cpErr != nil {
			flog.Error(fmt.Errorf("save checkpoint during resume run %d step %d: %w", runID, i, cpErr))
		}

		if err := e.executeStep(ctx, rc, step, runID, def.Name, true); err != nil {
			failed = true
			finalErr = err
			break
		}
	}

	e.finishRunRecord(ctx, runID, failed, finalErr)
	return finalErr
}

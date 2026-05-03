package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/flowline-io/flowbot/internal/store/model"
	otelattr "go.opentelemetry.io/otel/attribute"
)

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

// RunStore interface (add GetRun method)
type RunStore interface {
	CreateRun(pipelineName, eventID, eventType string) (*model.PipelineRun, error)
	UpdateRunStatus(runID int64, status model.PipelineState, errMsg string) error
	CreateStepRun(runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error)
	UpdateStepRun(stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(runID int64, data any) error
	GetIncompleteRuns() ([]*model.PipelineRun, error)
	GetCheckpoint(runID int64, target any) error
	GetRun(runID int64) (*model.PipelineRun, error)
	UpdateRunHeartbeat(runID int64) error
	HasConsumed(consumerName, eventID string) (bool, error)
	RecordConsumption(consumerName, eventID string) error
}

type Engine struct {
	defs    []Definition
	store   RunStore
	handler func(ctx context.Context, event types.DataEvent) error
}

func NewEngine(defs []Definition, store RunStore) *Engine {
	e := &Engine{
		defs:  defs,
		store: store,
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

	if e.store != nil {
		consumed, err := e.store.HasConsumed(def.Name, event.EventID)
		if err != nil {
			return fmt.Errorf("check consumption: %w", err)
		}
		if consumed {
			flog.Info("pipeline %s already consumed event %s", def.Name, event.EventID)
			return nil
		}
		if err := e.store.RecordConsumption(def.Name, event.EventID); err != nil {
			return fmt.Errorf("record consumption: %w", err)
		}
	}

	runID, err := e.createRunRecord(ctx, def.Name, event.EventID, event.EventType)
	if err != nil {
		return err
	}

	rc := NewRenderContext(event)
	failed := false
	var finalErr error

	for i, step := range def.Steps {
		// Save checkpoint before executing each step.
		if def.Resumable && e.store != nil && runID != 0 {
			cp := &CheckpointData{
				StepIndex:   i,
				StepResults: buildStepResults(rc),
				Event:       event,
				HeartbeatAt: time.Now(),
			}
			if cpErr := e.store.SaveCheckpoint(runID, cp); cpErr != nil {
				flog.Error(fmt.Errorf("save checkpoint pipeline %s step %d: %w", def.Name, i, cpErr))
			}
		}

		if err := e.executeStep(ctx, rc, step, runID, def.Name, def.Resumable); err != nil {
			failed = true
			finalErr = err
			break
		}
	}

	e.finishRunRecord(runID, failed, finalErr)

	if finalErr != nil {
		return finalErr
	}
	return nil
}

func (e *Engine) executeStep(ctx context.Context, rc *RenderContext, step Step, runID int64, pipelineName string, resumable bool) error {
	ctx, span := trace.StartSpan(ctx, "pipeline."+pipelineName+".step."+step.Name,
		otelattr.String("pipeline.step.name", step.Name),
		otelattr.String("pipeline.step.capability", string(step.Capability)),
		otelattr.String("pipeline.step.operation", step.Operation),
	)
	defer span.End()

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

	retryCfg := step.Retry
	bo := retryCfg.BuildBackOff()

	for {
		res, err := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
		if err == nil {
			stepResult := extractResult(res)
			rc.RecordStepResult(step.Name, stepResult)
			e.updateStepRunRecord(stepRunID, model.PipelineDone, stepResult, "", attempt)
			flog.Info("pipeline %s step %s completed (attempt %d)", pipelineName, step.Name, attempt)
			return nil
		}

		trace.RecordError(ctx, err)

		if !retryCfg.RetryEnabled() || !isRetryable(err, retryCfg) {
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, err.Error(), attempt)
			return fmt.Errorf("step %s: %w", step.Name, err)
		}

		nextDelay := bo.NextBackOff()
		if nextDelay == backoff.Stop {
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, err.Error(), attempt)
			return fmt.Errorf("step %s (retries exhausted): %w", step.Name, err)
		}

		flog.Info("pipeline %s step %s attempt %d failed, retrying in %v: %v",
			pipelineName, step.Name, attempt, nextDelay, err)

		select {
		case <-ctx.Done():
			e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, ctx.Err().Error(), attempt)
			return fmt.Errorf("step %s cancelled: %w", step.Name, ctx.Err())
		case <-time.After(nextDelay):
		}

		attempt++
	}
}

func isRetryable(err error, cfg *types.RetryConfig) bool {
	if cfg == nil || len(cfg.RetryOn) == 0 {
		return true
	}
	var te *types.Error
	if errors.As(err, &te) {
		if te.Retryable {
			return true
		}
	}
	for _, target := range cfg.RetryOn {
		if containsErrorCode(err, target) {
			return true
		}
	}
	return false
}

func containsErrorCode(err error, code string) bool {
	var te *types.Error
	if errors.As(err, &te) {
		if te.Code == code {
			return true
		}
	}
	// Walk wrapped errors.
	return false
}

func (e *Engine) heartbeatLoop(ctx context.Context, runID int64, pipelineName string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.store.UpdateRunHeartbeat(runID); err != nil {
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
	run, err := e.store.CreateRun(name, eventID, eventType)
	if err != nil {
		return 0, fmt.Errorf("create run: %w", err)
	}
	return run.ID, nil
}

func (e *Engine) createStepRunRecord(ctx context.Context, runID int64, stepName, capability, operation string, params map[string]any, attempt int) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	paramsJSON := model.JSON{}
	_ = paramsJSON.Scan(convertToTypesKV(params))
	sr, err := e.store.CreateStepRun(runID, stepName, capability, operation, paramsJSON, attempt)
	if err != nil {
		return 0, fmt.Errorf("create step run %s: %w", stepName, err)
	}
	return sr.ID, nil
}

func (e *Engine) updateStepRunRecord(stepRunID int64, status model.PipelineState, result map[string]any, errMsg string, attempt int) {
	if e.store == nil || stepRunID == 0 {
		return
	}
	var resultJSON model.JSON
	if result != nil {
		_ = resultJSON.Scan(convertToTypesKV(result))
	}
	_ = e.store.UpdateStepRun(stepRunID, status, resultJSON, errMsg, attempt)
}

func (e *Engine) finishRunRecord(runID int64, failed bool, finalErr error) {
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
	_ = e.store.UpdateRunStatus(runID, status, errMsg)
}

func extractResult(res *ability.InvokeResult) map[string]any {
	stepResult := map[string]any{}
	if res.Data == nil {
		return stepResult
	}
	if m, ok := res.Data.(map[string]any); ok {
		return m
	}
	dataJSON, _ := sonic.Marshal(res.Data)
	_ = sonic.Unmarshal(dataJSON, &stepResult)
	return stepResult
}

func convertToTypesKV(m map[string]any) types.KV {
	result := make(types.KV, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// ResumePipeline attempts to resume a pipeline run from its last checkpoint.
// It reloads the checkpoint, reconstructs the RenderContext, and continues
// from the checkpointed step index.
func (e *Engine) ResumePipeline(ctx context.Context, runID int64) error {
	if e.store == nil {
		return fmt.Errorf("pipeline store not available")
	}

	run, err := e.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("get run %d: %w", runID, err)
	}

	cp := &CheckpointData{}
	if err := e.store.GetCheckpoint(runID, cp); err != nil {
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
		if cpErr := e.store.SaveCheckpoint(runID, &CheckpointData{
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

	e.finishRunRecord(runID, failed, finalErr)
	return finalErr
}

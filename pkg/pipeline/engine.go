package pipeline

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/flowline-io/flowbot/internal/store/model"
)

type RunStore interface {
	CreateRun(pipelineName, eventID, eventType string) (*model.PipelineRun, error)
	UpdateRunStatus(runID int64, status model.PipelineState, errMsg string) error
	CreateStepRun(runID int64, stepName, capability, operation string, params model.JSON) (*model.PipelineStepRun, error)
	UpdateStepRun(stepRunID int64, status model.PipelineState, result model.JSON, errMsg string) error
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

	for _, step := range def.Steps {
		if err := e.executeStep(ctx, rc, step, runID, def.Name); err != nil {
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

func (e *Engine) executeStep(ctx context.Context, rc *RenderContext, step Step, runID int64, pipelineName string) error {
	renderedParams, err := rc.RenderParams(step.Params)
	if err != nil {
		return fmt.Errorf("render params step %s: %w", step.Name, err)
	}

	stepRunID, err := e.createStepRunRecord(ctx, runID, step.Name, string(step.Capability), step.Operation, renderedParams)
	if err != nil {
		return err
	}

	res, err := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
	if err != nil {
		e.updateStepRunRecord(stepRunID, model.PipelineCancel, nil, err.Error())
		return fmt.Errorf("step %s: %w", step.Name, err)
	}

	stepResult := extractResult(res)
	rc.RecordStepResult(step.Name, stepResult)

	e.updateStepRunRecord(stepRunID, model.PipelineDone, stepResult, "")

	flog.Info("pipeline %s step %s completed", pipelineName, step.Name)
	return nil
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

func (e *Engine) createStepRunRecord(ctx context.Context, runID int64, stepName, capability, operation string, params map[string]any) (int64, error) {
	if e.store == nil {
		return 0, nil
	}
	paramsJSON := model.JSON{}
	_ = paramsJSON.Scan(convertToTypesKV(params))
	sr, err := e.store.CreateStepRun(runID, stepName, capability, operation, paramsJSON)
	if err != nil {
		return 0, fmt.Errorf("create step run %s: %w", stepName, err)
	}
	return sr.ID, nil
}

func (e *Engine) updateStepRunRecord(stepRunID int64, status model.PipelineState, result map[string]any, errMsg string) {
	if e.store == nil || stepRunID == 0 {
		return
	}
	var resultJSON model.JSON
	if result != nil {
		_ = resultJSON.Scan(convertToTypesKV(result))
	}
	_ = e.store.UpdateStepRun(stepRunID, status, resultJSON, errMsg)
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

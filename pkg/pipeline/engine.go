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

	var runID int64
	if e.store != nil {
		run, err := e.store.CreateRun(def.Name, event.EventID, event.EventType)
		if err != nil {
			return fmt.Errorf("create run: %w", err)
		}
		runID = run.ID
	}

	rc := NewRenderContext(event)
	failed := false
	var finalErr error

	for _, step := range def.Steps {
		renderedParams, err := rc.RenderParams(step.Params)
		if err != nil {
			failed = true
			finalErr = fmt.Errorf("render params step %s: %w", step.Name, err)
			break
		}

		var stepRunID int64
		if e.store != nil {
			paramsJSON := model.JSON{}
			_ = paramsJSON.Scan(convertToTypesKV(renderedParams))
			sr, err := e.store.CreateStepRun(runID, step.Name, string(step.Capability), step.Operation, paramsJSON)
			if err != nil {
				failed = true
				finalErr = fmt.Errorf("create step run %s: %w", step.Name, err)
				break
			}
			stepRunID = sr.ID
		}

		res, err := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
		if err != nil {
			failed = true
			finalErr = fmt.Errorf("step %s: %w", step.Name, err)
			if e.store != nil {
				_ = e.store.UpdateStepRun(stepRunID, model.PipelineCancel, nil, err.Error())
			}
			break
		}

		stepResult := map[string]any{}
		if res.Data != nil {
			if m, ok := res.Data.(map[string]any); ok {
				stepResult = m
			} else {
				dataJSON, _ := sonic.Marshal(res.Data)
				_ = sonic.Unmarshal(dataJSON, &stepResult)
			}
		}
		rc.RecordStepResult(step.Name, stepResult)

		if e.store != nil {
			resultJSON := model.JSON{}
			_ = resultJSON.Scan(convertToTypesKV(stepResult))
			_ = e.store.UpdateStepRun(stepRunID, model.PipelineDone, resultJSON, "")
		}

		flog.Info("pipeline %s step %s completed", def.Name, step.Name)
	}

	if e.store != nil {
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

	if failed && finalErr != nil {
		return finalErr
	}

	return nil
}

func convertToTypesKV(m map[string]any) types.KV {
	result := make(types.KV, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

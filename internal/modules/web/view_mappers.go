package web

import (
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func mapPipelineDefinition(row *gen.PipelineDefinition) model.PipelineDefinition {
	if row == nil {
		return model.PipelineDefinition{}
	}
	return model.PipelineDefinition{
		ID:            row.ID,
		Name:          row.Name,
		Description:   row.Description,
		Status:        string(row.Status),
		Version:       row.Version,
		CreatedBy:     row.CreatedBy,
		YamlDraft:     row.YamlDraft,
		YamlPublished: row.YamlPublished,
	}
}

func mapPipelineDefinitions(rows []*gen.PipelineDefinition) []model.PipelineDefinition {
	out := make([]model.PipelineDefinition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineDefinition(row))
	}
	return out
}

func mapPipelineRun(row *gen.PipelineRun) model.PipelineRun {
	if row == nil {
		return model.PipelineRun{}
	}
	return model.PipelineRun{
		ID:            row.ID,
		PipelineName:  row.PipelineName,
		EventID:       row.EventID,
		EventType:     row.EventType,
		TriggerSource: string(row.TriggerSource),
		Status:        row.Status,
		Error:         row.Error,
		CreatedAt:     row.CreatedAt,
		StartedAt:     row.StartedAt,
		CompletedAt:   row.CompletedAt,
	}
}

func mapPipelineRuns(rows []*gen.PipelineRun) []model.PipelineRun {
	out := make([]model.PipelineRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineRun(row))
	}
	return out
}

func mapPipelineStepRun(row *gen.PipelineStepRun) model.PipelineStepRun {
	if row == nil {
		return model.PipelineStepRun{}
	}
	return model.PipelineStepRun{
		ID:          row.ID,
		StepName:    row.StepName,
		Capability:  row.Capability,
		Operation:   row.Operation,
		Attempt:     row.Attempt,
		Status:      row.Status,
		StartedAt:   row.StartedAt,
		CompletedAt: row.CompletedAt,
		Error:       row.Error,
		Params:      row.Params,
		Result:      row.Result,
	}
}

func mapPipelineStepRuns(rows []*gen.PipelineStepRun) []model.PipelineStepRun {
	out := make([]model.PipelineStepRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapPipelineStepRun(row))
	}
	return out
}

// mapPipelineStepRunsAny converts stored or prefetched step payloads into model rows.
func mapPipelineStepRunsAny(steps any) []model.PipelineStepRun {
	switch v := steps.(type) {
	case []model.PipelineStepRun:
		return v
	case []*gen.PipelineStepRun:
		return mapPipelineStepRuns(v)
	default:
		return nil
	}
}

func mapPipelineRunInfo(info store.PipelineRunInfo) model.PipelineRunInfo {
	return model.PipelineRunInfo{
		ID:            info.ID,
		PipelineName:  info.PipelineName,
		EventID:       info.EventID,
		Status:        info.Status,
		TriggerSource: info.TriggerSource,
	}
}

func mapPipelineRunInfoMap(runMap map[string][]store.PipelineRunInfo) map[string][]model.PipelineRunInfo {
	if runMap == nil {
		return nil
	}
	out := make(map[string][]model.PipelineRunInfo, len(runMap))
	for eventID, infos := range runMap {
		mapped := make([]model.PipelineRunInfo, 0, len(infos))
		for _, info := range infos {
			mapped = append(mapped, mapPipelineRunInfo(info))
		}
		out[eventID] = mapped
	}
	return out
}

func mapWorkflow(row *gen.Workflow) model.Workflow {
	if row == nil {
		return model.Workflow{}
	}
	return model.Workflow{
		ID:             row.ID,
		Name:           row.Name,
		Describe:       row.Describe,
		Enabled:        row.Enabled,
		Resumable:      row.Resumable,
		MaxConcurrency: row.MaxConcurrency,
		Pipeline:       append([]string(nil), row.Pipeline...),
		Inputs:         row.Inputs,
	}
}

func mapWorkflows(rows []*gen.Workflow) []model.Workflow {
	out := make([]model.Workflow, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflow(row))
	}
	return out
}

func mapWorkflowTrigger(row *gen.WorkflowTrigger) model.WorkflowTrigger {
	if row == nil {
		return model.WorkflowTrigger{}
	}
	return model.WorkflowTrigger{
		ID:         row.ID,
		WorkflowID: row.WorkflowID,
		Type:       row.Type,
		Enabled:    row.Enabled,
		Rule:       row.Rule,
	}
}

func mapWorkflowTriggers(rows []*gen.WorkflowTrigger) []model.WorkflowTrigger {
	out := make([]model.WorkflowTrigger, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflowTrigger(row))
	}
	return out
}

func mapWorkflowRun(row *gen.WorkflowRun) model.WorkflowRun {
	if row == nil {
		return model.WorkflowRun{}
	}
	return model.WorkflowRun{
		ID:           row.ID,
		WorkflowID:   row.WorkflowID,
		WorkflowName: row.WorkflowName,
		Status:       row.Status,
		TriggerType:  row.TriggerType,
		StartedAt:    row.StartedAt,
		CreatedAt:    row.CreatedAt,
		CompletedAt:  row.CompletedAt,
		Error:        row.Error,
	}
}

func mapWorkflowRuns(rows []*gen.WorkflowRun) []model.WorkflowRun {
	out := make([]model.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflowRun(row))
	}
	return out
}

func mapWorkflowStepRun(row *gen.WorkflowStepRun) model.WorkflowStepRun {
	if row == nil {
		return model.WorkflowStepRun{}
	}
	return model.WorkflowStepRun{
		ID:          row.ID,
		StepID:      row.StepID,
		StepName:    row.StepName,
		Action:      row.Action,
		ActionType:  row.ActionType,
		Attempt:     row.Attempt,
		Status:      row.Status,
		StartedAt:   row.StartedAt,
		CompletedAt: row.CompletedAt,
		Error:       row.Error,
		Params:      row.Params,
		Result:      row.Result,
	}
}

func mapWorkflowStepRuns(rows []*gen.WorkflowStepRun) []model.WorkflowStepRun {
	out := make([]model.WorkflowStepRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapWorkflowStepRun(row))
	}
	return out
}

func mapDataEvent(row *gen.DataEvent) model.DataEvent {
	if row == nil {
		return model.DataEvent{}
	}
	return model.DataEvent{
		EventID:    row.EventID,
		EventType:  row.EventType,
		Source:     row.Source,
		Capability: row.Capability,
		EntityID:   row.EntityID,
		CreatedAt:  row.CreatedAt,
		Data:       row.Data,
	}
}

func mapDataEvents(rows []*gen.DataEvent) []model.DataEvent {
	out := make([]model.DataEvent, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapDataEvent(row))
	}
	return out
}


func mapPermissionsView(view chatagent.PermissionsView) model.PermissionsView {
	return model.PermissionsView{
		Defaults: view.Defaults,
		User:     view.User,
	}
}

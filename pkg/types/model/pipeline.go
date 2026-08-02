package model

import "time"

// PipelineDefinition is a pipeline definition row for UI and engine catalog use.
type PipelineDefinition struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	Status        string  `json:"status"`
	Version       int     `json:"version,omitempty"`
	CreatedBy     string  `json:"created_by,omitempty"`
	YamlDraft     string  `json:"yaml_draft,omitempty"`
	YamlPublished *string `json:"yaml_published,omitempty"`
}

// PipelineRun is a pipeline run row for UI and engine persistence.
type PipelineRun struct {
	ID            int64      `json:"id"`
	PipelineName  string     `json:"pipeline_name,omitempty"`
	EventID       string     `json:"event_id"`
	EventType     string     `json:"event_type,omitempty"`
	TriggerSource string     `json:"trigger_source,omitempty"`
	Status        int        `json:"status"`
	Error         string     `json:"error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// PipelineStepRun is a pipeline step run row for UI and engine persistence.
type PipelineStepRun struct {
	ID          int64          `json:"id,omitempty"`
	StepName    string         `json:"step_name"`
	Capability  string         `json:"capability,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	Status      int            `json:"status"`
	Attempt     int            `json:"attempt,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
}

// PipelineRunInfo links a data event to a matching pipeline run for UI display.
type PipelineRunInfo struct {
	ID            int64  `json:"id"`
	PipelineName  string `json:"pipeline_name"`
	EventID       string `json:"event_id"`
	Status        string `json:"status"`
	TriggerSource string `json:"trigger_source,omitempty"`
}

// ResourceLink records a capability resource edge created during a pipeline step.
type ResourceLink struct {
	SourceEventID    string `json:"source_event_id"`
	TargetEventID    string `json:"target_event_id"`
	SourceApp        string `json:"source_app,omitempty"`
	TargetApp        string `json:"target_app,omitempty"`
	SourceCapability string `json:"source_capability,omitempty"`
	TargetCapability string `json:"target_capability,omitempty"`
	SourceEntityID   string `json:"source_entity_id,omitempty"`
	TargetEntityID   string `json:"target_entity_id,omitempty"`
	PipelineRunID    int64  `json:"pipeline_run_id,omitempty"`
	PipelineName     string `json:"pipeline_name,omitempty"`
}

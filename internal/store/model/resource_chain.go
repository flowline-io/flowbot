package model

import "time"

type ResourceLink struct {
	ID               int64     `json:"id"`
	SourceEventID    string    `json:"source_event_id"`
	TargetEventID    string    `json:"target_event_id"`
	SourceApp        string    `json:"source_app"`
	TargetApp        string    `json:"target_app"`
	SourceCapability string    `json:"source_capability"`
	TargetCapability string    `json:"target_capability"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetEntityID   string    `json:"target_entity_id"`
	PipelineRunID    int64     `json:"pipeline_run_id,omitzero"`
	PipelineName     string    `json:"pipeline_name"`
	CreatedAt        time.Time `json:"created_at"`
}

type ResourceRelations struct {
	App        string        `json:"app"`
	EntityID   string        `json:"entity_id"`
	Upstream   []ResourceRef `json:"upstream"`
	Downstream []ResourceRef `json:"downstream"`
}

type ResourceRef struct {
	App          string `json:"app"`
	EntityID     string `json:"entity_id"`
	Capability   string `json:"capability,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
}

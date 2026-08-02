package types

import "time"

// ResourceRef identifies a resource by app, entity_id, and optional metadata.
type ResourceRef struct {
	App          string `json:"app"`
	EntityID     string `json:"entity_id"`
	Capability   string `json:"capability,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
}

// ResourceEdge represents a directed resource link with full source and target
// details plus pipeline metadata and creation time.
type ResourceEdge struct {
	SourceApp        string    `json:"source_app"`
	SourceCapability string    `json:"source_capability"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetApp        string    `json:"target_app"`
	TargetCapability string    `json:"target_capability"`
	TargetEntityID   string    `json:"target_entity_id"`
	PipelineName     string    `json:"pipeline_name"`
	CreatedAt        time.Time `json:"created_at"`
}

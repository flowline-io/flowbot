package types

import "time"

// ResourceLinkRunKind names the execution surface that produced a resource_links row.
type ResourceLinkRunKind string

const (
	// ResourceLinkRunKindPipeline is a pipeline step edge.
	ResourceLinkRunKindPipeline ResourceLinkRunKind = "pipeline"
	// ResourceLinkRunKindWorkflow is a workflow step edge.
	ResourceLinkRunKindWorkflow ResourceLinkRunKind = "workflow"
	// ResourceLinkRunKindFunction is a named-function invoke edge.
	ResourceLinkRunKindFunction ResourceLinkRunKind = "function"
	// ResourceLinkRunKindMCP is an MCP mutation external-input edge.
	ResourceLinkRunKindMCP ResourceLinkRunKind = "mcp"
	// ResourceLinkRunKindManual is a manually recorded edge.
	ResourceLinkRunKindManual ResourceLinkRunKind = "manual"
)

// ResourceLinkStatus is the outcome recorded on a resource_links row.
type ResourceLinkStatus string

const (
	// ResourceLinkStatusOK is a successful edge.
	ResourceLinkStatusOK ResourceLinkStatus = "ok"
	// ResourceLinkStatusFailed is a failed step edge (no target entity).
	ResourceLinkStatusFailed ResourceLinkStatus = "failed"
)

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

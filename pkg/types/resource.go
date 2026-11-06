package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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

// ResourceLinkFailTargetEventID builds the synthetic target_event_id for a failed step edge.
// It is not a data_events row; fail: targets are lineage leaves.
func ResourceLinkFailTargetEventID(runID int64, stepName string) string {
	return fmt.Sprintf("fail:%d:%s", runID, strings.TrimSpace(stepName))
}

// ResourceLinkMCPAgentApp is the flow-graph source.app for an MCP external-input edge (D11).
// Tokens have no display name; the access-token row id is the stable node id.
func ResourceLinkMCPAgentApp(tokenID int64) string {
	return "agent:" + strconv.FormatInt(tokenID, 10)
}

// ResourceLinkMCPSourceEventID is the synthetic source_event_id for an MCP external-input edge.
func ResourceLinkMCPSourceEventID(requestID string) string {
	return "mcp:" + strings.TrimSpace(requestID)
}

// ResourceLinkWorkflowSourceEventID is the synthetic source_event_id for a workflow run edge.
func ResourceLinkWorkflowSourceEventID(runID int64) string {
	return fmt.Sprintf("wf:%d", runID)
}

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

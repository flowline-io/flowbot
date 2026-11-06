package types

import (
	"cmp"
	"fmt"
	"slices"
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
	RunKind          string    `json:"run_kind,omitempty"`
	Status           string    `json:"status,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Flow graph and lineage trace window / hop limits.
const (
	// FlowGraphDefaultWindow is the default lookback when since is omitted.
	FlowGraphDefaultWindow = 7 * 24 * time.Hour
	// FlowGraphMaxWindowDays caps any flow.graph window (also when retention is unlimited).
	FlowGraphMaxWindowDays = 90
	// LineageTraceDefaultDepth is the default hop depth for lineage.trace.
	LineageTraceDefaultDepth = 10
	// LineageTraceMaxDepth is the hard hop depth ceiling.
	LineageTraceMaxDepth = 10
	// LineageTraceMaxNodes is the hard node count ceiling for lineage.trace.
	LineageTraceMaxNodes = 500
)

// LineageTraceDirection names which way a multi-hop trace walks edges.
type LineageTraceDirection string

const (
	// LineageTraceDirectionUp walks toward sources (default: "where did this come from").
	LineageTraceDirectionUp LineageTraceDirection = "up"
	// LineageTraceDirectionDown walks toward targets.
	LineageTraceDirectionDown LineageTraceDirection = "down"
	// LineageTraceDirectionBoth walks both directions.
	LineageTraceDirectionBoth LineageTraceDirection = "both"
)

// FlowGraphVia identifies the execution surface that produced an aggregated edge.
type FlowGraphVia struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// FlowGraphEdge is one aggregated directed edge in a flow.graph response.
type FlowGraphEdge struct {
	Source       string       `json:"source"`
	Target       string       `json:"target"`
	Via          FlowGraphVia `json:"via"`
	CountOK      int64        `json:"count_ok"`
	CountFailed  int64        `json:"count_failed"`
	LastAt       *time.Time   `json:"last_at,omitempty"`
	Static       bool         `json:"static"`
}

// FlowGraphNode is an app (or agent:*) vertex in a flow.graph response.
type FlowGraphNode struct {
	ID string `json:"id"`
}

// FlowGraphWindow is the resolved time range for a flow.graph query.
type FlowGraphWindow struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

// FlowGraph is the aggregated topology returned by flow.graph.
type FlowGraph struct {
	Nodes  []FlowGraphNode `json:"nodes"`
	Edges  []FlowGraphEdge `json:"edges"`
	Window FlowGraphWindow `json:"window"`
}

// FlowGraphQuery is the input for live resource_links aggregation.
type FlowGraphQuery struct {
	Since                    time.Time
	Until                    time.Time
	App                      string
	IncludeStatic            bool // accepted for MCP compatibility; ignored until static edges land
	DataEventsRetentionDays  int
}

// LineageTraceQuery seeds a multi-hop lineage walk.
// Exactly one of (App+EntityID), EventID, or CorrelationID must be set.
type LineageTraceQuery struct {
	App           string
	EntityID      string
	EventID       string
	CorrelationID string
	Direction     LineageTraceDirection
	Depth         int
}

// LineageTraceNode is one resource or synthetic leaf in a lineage.trace response.
type LineageTraceNode struct {
	App        string `json:"app,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`
	EventID    string `json:"event_id,omitempty"`
	Capability string `json:"capability,omitempty"`
}

// LineageTraceEdge is one hop in a lineage.trace response.
type LineageTraceEdge struct {
	SourceEventID    string    `json:"source_event_id"`
	TargetEventID    string    `json:"target_event_id"`
	SourceApp        string    `json:"source_app"`
	TargetApp        string    `json:"target_app"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetEntityID   string    `json:"target_entity_id"`
	SourceCapability string    `json:"source_capability,omitempty"`
	TargetCapability string    `json:"target_capability,omitempty"`
	RunKind          string    `json:"run_kind"`
	Status           string    `json:"status"`
	ViaName          string    `json:"via_name,omitempty"`
	CorrelationID    string    `json:"correlation_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	Depth            int       `json:"depth"`
}

// LineageTrace is the multi-hop result for lineage.trace.
type LineageTrace struct {
	Nodes     []LineageTraceNode `json:"nodes"`
	Edges     []LineageTraceEdge `json:"edges"`
	Truncated bool               `json:"truncated"`
}

// LineageSearchQuery is the input for lineage.search (SearchNodes).
type LineageSearchQuery struct {
	Query  string
	Limit  int
	Cursor string
}

// LineageSearchResult is the paged node list for lineage.search.
type LineageSearchResult struct {
	Items      []ResourceRef `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// EventsTimelineQuery is the input for events.timeline.
type EventsTimelineQuery struct {
	Since          time.Time
	Until          time.Time
	App            string
	EventType      string
	CorrelationID  string
	Limit          int
	Cursor         string
	IncludePayload bool
	// CallerUID restricts rows to this uid when non-empty (non-admin callers).
	CallerUID string
	// Scopes is the caller's token scopes; used only when IncludePayload is true.
	Scopes []string
}

// EventsTimelineItem is one data_events row in an events.timeline response.
type EventsTimelineItem struct {
	EventID         string         `json:"event_id"`
	EventType       string         `json:"event_type"`
	Source          string         `json:"source,omitempty"`
	App             string         `json:"app,omitempty"`
	Capability      string         `json:"capability,omitempty"`
	Operation       string         `json:"operation,omitempty"`
	EntityID        string         `json:"entity_id,omitempty"`
	UID             string         `json:"uid,omitempty"`
	CorrelationID   string         `json:"correlation_id,omitempty"`
	CausationID     string         `json:"causation_id,omitempty"`
	TraceID         string         `json:"trace_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	Data            map[string]any `json:"data,omitempty"`
	PayloadOmitted  bool           `json:"payload_omitted,omitempty"`
}

// EventsTimelineResult is the paged list for events.timeline.
type EventsTimelineResult struct {
	Items      []EventsTimelineItem `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// NormalizeFlowGraphWindow resolves since/until with default 7d lookback and
// caps the span by min(data_events_days if >0, FlowGraphMaxWindowDays).
func NormalizeFlowGraphWindow(since, until time.Time, dataEventsRetentionDays int, now time.Time) (normalizedSince, normalizedUntil time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if until.IsZero() {
		normalizedUntil = now
	} else {
		normalizedUntil = until.UTC()
	}
	if since.IsZero() {
		normalizedSince = normalizedUntil.Add(-FlowGraphDefaultWindow)
	} else {
		normalizedSince = since.UTC()
	}
	if !normalizedSince.Before(normalizedUntil) {
		normalizedSince = normalizedUntil.Add(-FlowGraphDefaultWindow)
	}
	maxDays := FlowGraphMaxWindowDays
	if dataEventsRetentionDays > 0 && dataEventsRetentionDays < maxDays {
		maxDays = dataEventsRetentionDays
	}
	maxSpan := time.Duration(maxDays) * 24 * time.Hour
	if normalizedUntil.Sub(normalizedSince) > maxSpan {
		normalizedSince = normalizedUntil.Add(-maxSpan)
	}
	return normalizedSince, normalizedUntil
}

// NormalizeLineageTraceDirection returns a valid direction; empty defaults to up.
func NormalizeLineageTraceDirection(d LineageTraceDirection) LineageTraceDirection {
	switch d {
	case LineageTraceDirectionDown, LineageTraceDirectionBoth:
		return d
	default:
		return LineageTraceDirectionUp
	}
}

// NormalizeLineageTraceDepth clamps depth to [1, LineageTraceMaxDepth]; zero uses the default.
func NormalizeLineageTraceDepth(depth int) int {
	if depth <= 0 {
		return LineageTraceDefaultDepth
	}
	if depth > LineageTraceMaxDepth {
		return LineageTraceMaxDepth
	}
	return depth
}

// CapFlowGraph keeps at most maxEdges (failed first, then highest traffic) and
// at most maxNodes by incident degree. Nil graph yields an empty graph.
func CapFlowGraph(graph *FlowGraph, maxNodes, maxEdges int) *FlowGraph {
	if graph == nil {
		return &FlowGraph{Nodes: []FlowGraphNode{}, Edges: []FlowGraphEdge{}}
	}
	edges := capFlowEdges(graph.Edges, maxEdges)
	keep := capFlowNodeIDs(edges, maxNodes)
	filtered := make([]FlowGraphEdge, 0, len(edges))
	for _, e := range edges {
		if keep[e.Source] && keep[e.Target] {
			filtered = append(filtered, e)
		}
	}
	nodes := make([]FlowGraphNode, 0, len(keep))
	for id := range keep {
		nodes = append(nodes, FlowGraphNode{ID: id})
	}
	slices.SortFunc(nodes, func(a, b FlowGraphNode) int {
		return strings.Compare(a.ID, b.ID)
	})
	return &FlowGraph{Nodes: nodes, Edges: filtered, Window: graph.Window}
}

func capFlowEdges(edges []FlowGraphEdge, maxEdges int) []FlowGraphEdge {
	out := append([]FlowGraphEdge(nil), edges...)
	if maxEdges <= 0 || len(out) <= maxEdges {
		return out
	}
	slices.SortFunc(out, cmpFlowEdgePriority)
	return out[:maxEdges]
}

func cmpFlowEdgePriority(a, b FlowGraphEdge) int {
	aFail := a.CountFailed > 0
	bFail := b.CountFailed > 0
	if aFail != bFail {
		if aFail {
			return -1
		}
		return 1
	}
	if c := cmp.Compare(b.CountOK+b.CountFailed, a.CountOK+a.CountFailed); c != 0 {
		return c
	}
	if c := strings.Compare(a.Source, b.Source); c != 0 {
		return c
	}
	return strings.Compare(a.Target, b.Target)
}

func capFlowNodeIDs(edges []FlowGraphEdge, maxNodes int) map[string]bool {
	degree := map[string]int64{}
	for _, e := range edges {
		w := e.CountOK + e.CountFailed
		if e.CountFailed > 0 {
			w += 1_000_000
		}
		degree[e.Source] += w
		degree[e.Target] += w
	}
	ids := make([]string, 0, len(degree))
	for id := range degree {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int {
		if c := cmp.Compare(degree[b], degree[a]); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
	if maxNodes > 0 && len(ids) > maxNodes {
		ids = ids[:maxNodes]
	}
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	return keep
}

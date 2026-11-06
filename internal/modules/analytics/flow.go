package analytics

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	appcfg "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

// FlowGraph returns live-aggregated resource_links edges for the requested window.
// IncludeStatic is accepted for forward compatibility and ignored in this slice.
func (*Service) FlowGraph(ctx context.Context, q types.FlowGraphQuery) (*types.FlowGraph, error) {
	retentionDays := q.DataEventsRetentionDays
	if retentionDays == 0 {
		retentionDays = appcfg.App.Retention.DataEventsDays
	}
	since, until := types.NormalizeFlowGraphWindow(q.Since, q.Until, retentionDays, time.Now().UTC())

	rc := store.NewResourceChainStore(store.ClientFromDB())
	edges, err := rc.AggregateFlowGraph(ctx, since, until, q.App)
	if err != nil {
		return nil, fmt.Errorf("analytics: flow graph: %w", err)
	}
	if edges == nil {
		edges = []types.FlowGraphEdge{}
	}

	seen := map[string]bool{}
	var nodes []types.FlowGraphNode
	addNode := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		nodes = append(nodes, types.FlowGraphNode{ID: id})
	}
	for _, edge := range edges {
		addNode(edge.Source)
		addNode(edge.Target)
	}
	slices.SortFunc(nodes, func(a, b types.FlowGraphNode) int {
		return strings.Compare(a.ID, b.ID)
	})

	return &types.FlowGraph{
		Nodes:  nodes,
		Edges:  edges,
		Window: types.FlowGraphWindow{Since: since, Until: until},
	}, nil
}

// FlowGraphUI returns a FlowGraph capped for the Analytics Flows web chart
// (prefer failed edges, then highest traffic) so 20 nodes / 50 edges stay interactive.
func (s *Service) FlowGraphUI(ctx context.Context, q types.FlowGraphQuery) (*types.FlowGraph, error) {
	graph, err := s.FlowGraph(ctx, q)
	if err != nil {
		return nil, err
	}
	return types.CapFlowGraph(graph, 20, 50), nil
}

// Trace returns a bounded multi-hop lineage walk from the given seed.
func (*Service) Trace(ctx context.Context, q types.LineageTraceQuery) (*types.LineageTrace, error) {
	if err := validateTraceQuery(q); err != nil {
		return nil, err
	}
	rc := store.NewResourceChainStore(store.ClientFromDB())
	out, err := rc.Trace(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("analytics: lineage trace: %w", err)
	}
	if out == nil {
		return &types.LineageTrace{Nodes: []types.LineageTraceNode{}, Edges: []types.LineageTraceEdge{}}, nil
	}
	return out, nil
}

func validateTraceQuery(q types.LineageTraceQuery) error {
	hasEntity := strings.TrimSpace(q.App) != "" && strings.TrimSpace(q.EntityID) != ""
	hasEvent := strings.TrimSpace(q.EventID) != ""
	hasCorr := strings.TrimSpace(q.CorrelationID) != ""
	n := 0
	if hasEntity {
		n++
	}
	if hasEvent {
		n++
	}
	if hasCorr {
		n++
	}
	if n != 1 {
		return &types.Error{
			Kind:    types.ErrInvalidArgument,
			Message: "trace requires exactly one of app+entity_id, event_id, or correlation_id",
			Field:   "seed",
		}
	}
	return nil
}

// Search returns distinct resource nodes matching query (lineage.search).
func (*Service) Search(ctx context.Context, q types.LineageSearchQuery) (*types.LineageSearchResult, error) {
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, &types.Error{Kind: types.ErrInvalidArgument, Message: "query is required", Field: "query"}
	}
	rc := store.NewResourceChainStore(store.ClientFromDB())
	items, next, err := rc.SearchNodes(ctx, query, q.Limit, q.Cursor)
	if err != nil {
		return nil, fmt.Errorf("analytics: lineage search: %w", err)
	}
	if items == nil {
		items = []types.ResourceRef{}
	}
	return &types.LineageSearchResult{Items: items, NextCursor: next}, nil
}

// Timeline returns a paged data_events window (events.timeline).
// Rows appear under lineage:read (+ uid for non-admins). Payload bodies follow D10:
// omitted by default with payload_omitted set; include_payload attaches data only
// when the caller holds service:{capability|app}:read (or admin:*), otherwise
// payload_omitted stays set. When include_payload is true and app is set but the
// caller cannot read that app, the whole call is permission_denied.
func (*Service) Timeline(ctx context.Context, q types.EventsTimelineQuery) (*types.EventsTimelineResult, error) {
	if q.IncludePayload && strings.TrimSpace(q.App) != "" {
		if !auth.HasMinimumServiceScope(q.Scopes, strings.TrimSpace(q.App), "GET") {
			return nil, types.Errorf(types.ErrForbidden, "include_payload requires read scope for app %s", q.App)
		}
	}
	since, until := types.NormalizeFlowGraphWindow(q.Since, q.Until, 0, time.Now().UTC())
	opts := store.ListDataEventsOptions{
		App:           strings.TrimSpace(q.App),
		EventType:     strings.TrimSpace(q.EventType),
		CorrelationID: strings.TrimSpace(q.CorrelationID),
		UID:           strings.TrimSpace(q.CallerUID),
		Limit:         q.Limit,
		Cursor:        q.Cursor,
		TimeStart:     &since,
		TimeEnd:       &until,
	}
	es := store.EventStoreFromDB()
	if es == nil {
		return &types.EventsTimelineResult{Items: []types.EventsTimelineItem{}}, nil
	}
	rows, next, err := es.ListDataEvents(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("analytics: events timeline: %w", err)
	}
	items := make([]types.EventsTimelineItem, 0, len(rows))
	for _, row := range rows {
		item := types.EventsTimelineItem{
			EventID:       row.EventID,
			EventType:     row.EventType,
			Source:        row.Source,
			App:           row.App,
			Capability:    row.Capability,
			Operation:     row.Operation,
			EntityID:      row.EntityID,
			UID:           row.UID,
			CorrelationID: row.CorrelationID,
			CausationID:   row.CausationID,
			TraceID:       row.TraceID,
			CreatedAt:     row.CreatedAt,
		}
		if q.IncludePayload {
			group := timelinePayloadGroup(row.Capability, row.App)
			if group != "" && auth.HasMinimumServiceScope(q.Scopes, group, "GET") {
				item.Data = row.Data
			} else {
				item.PayloadOmitted = true
			}
		} else {
			item.PayloadOmitted = true
		}
		items = append(items, item)
	}
	return &types.EventsTimelineResult{Items: items, NextCursor: next}, nil
}

func timelinePayloadGroup(capability, app string) string {
	if g := strings.TrimSpace(capability); g != "" {
		return g
	}
	return strings.TrimSpace(app)
}

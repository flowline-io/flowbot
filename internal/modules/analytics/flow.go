package analytics

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
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
		return errors.New("analytics: trace requires exactly one of app+entity_id, event_id, or correlation_id")
	}
	return nil
}

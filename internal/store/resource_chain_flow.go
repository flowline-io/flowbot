package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AggregateFlowGraph groups resource_links in [since, until] by
// (source_app, target_app, run_kind, pipeline_name) and returns aggregated edges.
// App, when non-empty, keeps rows where source_app or target_app matches.
func (s *ResourceChainStore) AggregateFlowGraph(ctx context.Context, since, until time.Time, app string) ([]types.FlowGraphEdge, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	q := s.client.ResourceLink.Query().Where(
		resourcelink.CreatedAtGTE(since),
		resourcelink.CreatedAtLTE(until),
	)
	app = strings.TrimSpace(app)
	if app != "" {
		q = q.Where(resourcelink.Or(
			resourcelink.SourceAppEQ(app),
			resourcelink.TargetAppEQ(app),
		))
	}

	// Select only aggregation columns and fold in Go so SQLite/Postgres Max(time)
	// scan differences do not matter; the (source_app, target_app, created_at) index
	// keeps the window scan on the hot path.
	links, err := q.Select(
		resourcelink.FieldSourceApp,
		resourcelink.FieldTargetApp,
		resourcelink.FieldRunKind,
		resourcelink.FieldPipelineName,
		resourcelink.FieldStatus,
		resourcelink.FieldCreatedAt,
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("aggregate flow graph: %w", err)
	}
	return foldFlowGraphEdges(links), nil
}

type flowEdgeKey struct {
	source, target, kind, name string
}

func foldFlowGraphEdges(links []*gen.ResourceLink) []types.FlowGraphEdge {
	merged := map[flowEdgeKey]*types.FlowGraphEdge{}
	for _, row := range links {
		key := flowEdgeKey{row.SourceApp, row.TargetApp, string(row.RunKind), row.PipelineName}
		edge, ok := merged[key]
		if !ok {
			edge = &types.FlowGraphEdge{
				Source: row.SourceApp,
				Target: row.TargetApp,
				Via: types.FlowGraphVia{
					Kind: string(row.RunKind),
					Name: row.PipelineName,
				},
				Static: false,
			}
			merged[key] = edge
		}
		if row.Status == resourcelink.StatusFailed {
			edge.CountFailed++
		} else {
			edge.CountOK++
		}
		created := row.CreatedAt.UTC()
		if edge.LastAt == nil || created.After(*edge.LastAt) {
			edge.LastAt = &created
		}
	}
	out := make([]types.FlowGraphEdge, 0, len(merged))
	for _, edge := range merged {
		out = append(out, *edge)
	}
	slices.SortFunc(out, func(a, b types.FlowGraphEdge) int {
		if c := strings.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		if c := strings.Compare(a.Target, b.Target); c != 0 {
			return c
		}
		if c := strings.Compare(a.Via.Kind, b.Via.Kind); c != 0 {
			return c
		}
		return strings.Compare(a.Via.Name, b.Via.Name)
	})
	return out
}

type traceFrontier struct {
	eventID string
	depth   int
}

type traceWalkState struct {
	out      *types.LineageTrace
	nodeInfo map[string]types.LineageTraceNode
	visited  map[string]bool
	seenEdge map[string]bool
	queue    []traceFrontier
}

// Trace walks resource_links from a seed with bounded depth and node count.
func (s *ResourceChainStore) Trace(ctx context.Context, q types.LineageTraceQuery) (*types.LineageTrace, error) {
	if s == nil || s.client == nil {
		return &types.LineageTrace{Nodes: []types.LineageTraceNode{}, Edges: []types.LineageTraceEdge{}}, nil
	}
	direction := types.NormalizeLineageTraceDirection(q.Direction)
	maxDepth := types.NormalizeLineageTraceDepth(q.Depth)

	seeds, err := s.traceSeeds(ctx, q)
	if err != nil {
		return nil, err
	}
	state := newTraceWalkState(seeds)
	if len(state.queue) == 0 {
		return state.out, nil
	}
	if err := s.runTraceWalk(ctx, state, direction, maxDepth); err != nil {
		return nil, err
	}
	finalizeTraceNodes(state)
	return state.out, nil
}

func newTraceWalkState(seeds []string) *traceWalkState {
	state := &traceWalkState{
		out: &types.LineageTrace{
			Nodes: []types.LineageTraceNode{},
			Edges: []types.LineageTraceEdge{},
		},
		nodeInfo: map[string]types.LineageTraceNode{},
		visited:  map[string]bool{},
		seenEdge: map[string]bool{},
	}
	for _, id := range seeds {
		id = strings.TrimSpace(id)
		if id == "" || state.visited[id] {
			continue
		}
		if !state.tryVisit(id) {
			break
		}
		state.queue = append(state.queue, traceFrontier{eventID: id, depth: 0})
	}
	return state
}

func (state *traceWalkState) tryVisit(id string) bool {
	if len(state.visited) >= types.LineageTraceMaxNodes {
		state.out.Truncated = true
		return false
	}
	state.visited[id] = true
	return true
}

func (s *ResourceChainStore) runTraceWalk(ctx context.Context, state *traceWalkState, direction types.LineageTraceDirection, maxDepth int) error {
	for len(state.queue) > 0 {
		cur := state.queue[0]
		state.queue = state.queue[1:]
		if cur.depth >= maxDepth {
			state.out.Truncated = true
			continue
		}
		links, err := s.traceNeighborLinks(ctx, cur.eventID, direction)
		if err != nil {
			return err
		}
		for _, link := range links {
			state.recordEdge(link, cur.depth+1)
			for _, next := range traceNextEventIDs(cur.eventID, link, direction) {
				if next == "" || state.visited[next] {
					continue
				}
				if !state.tryVisit(next) {
					continue
				}
				state.queue = append(state.queue, traceFrontier{eventID: next, depth: cur.depth + 1})
			}
		}
	}
	return nil
}

func (state *traceWalkState) recordEdge(link *gen.ResourceLink, depth int) {
	edgeKey := link.SourceEventID + ">" + link.TargetEventID
	if state.seenEdge[edgeKey] {
		return
	}
	state.seenEdge[edgeKey] = true
	state.out.Edges = append(state.out.Edges, types.LineageTraceEdge{
		SourceEventID:    link.SourceEventID,
		TargetEventID:    link.TargetEventID,
		SourceApp:        link.SourceApp,
		TargetApp:        link.TargetApp,
		SourceEntityID:   link.SourceEntityID,
		TargetEntityID:   link.TargetEntityID,
		SourceCapability: link.SourceCapability,
		TargetCapability: link.TargetCapability,
		RunKind:          string(link.RunKind),
		Status:           string(link.Status),
		ViaName:          link.PipelineName,
		CorrelationID:    link.CorrelationID,
		CreatedAt:        link.CreatedAt.UTC(),
		Depth:            depth,
	})
	rememberTraceEndpoint(state.nodeInfo, link.SourceEventID, link.SourceApp, link.SourceEntityID, link.SourceCapability)
	rememberTraceEndpoint(state.nodeInfo, link.TargetEventID, link.TargetApp, link.TargetEntityID, link.TargetCapability)
}

func finalizeTraceNodes(state *traceWalkState) {
	for id := range state.visited {
		if n, ok := state.nodeInfo[id]; ok {
			state.out.Nodes = append(state.out.Nodes, n)
			continue
		}
		state.out.Nodes = append(state.out.Nodes, types.LineageTraceNode{EventID: id})
	}
	slices.SortFunc(state.out.Nodes, func(a, b types.LineageTraceNode) int {
		if c := strings.Compare(a.EventID, b.EventID); c != 0 {
			return c
		}
		if c := strings.Compare(a.App, b.App); c != 0 {
			return c
		}
		return strings.Compare(a.EntityID, b.EntityID)
	})
	slices.SortFunc(state.out.Edges, func(a, b types.LineageTraceEdge) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.SourceEventID+a.TargetEventID, b.SourceEventID+b.TargetEventID)
	})
}

func (s *ResourceChainStore) traceSeeds(ctx context.Context, q types.LineageTraceQuery) ([]string, error) {
	app := strings.TrimSpace(q.App)
	entityID := strings.TrimSpace(q.EntityID)
	eventID := strings.TrimSpace(q.EventID)
	correlationID := strings.TrimSpace(q.CorrelationID)

	switch {
	case eventID != "":
		return []string{eventID}, nil
	case correlationID != "":
		return s.traceSeedsByCorrelation(ctx, correlationID)
	case app != "" && entityID != "":
		return s.traceSeedsByEntity(ctx, app, entityID)
	default:
		return nil, errors.New("trace: require event_id, correlation_id, or app+entity_id")
	}
}

func (s *ResourceChainStore) traceSeedsByCorrelation(ctx context.Context, correlationID string) ([]string, error) {
	links, err := s.client.ResourceLink.Query().
		Where(resourcelink.CorrelationIDEQ(correlationID)).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("trace seeds by correlation: %w", err)
	}
	return uniqueEventIDsFromLinks(links, nil), nil
}

func (s *ResourceChainStore) traceSeedsByEntity(ctx context.Context, app, entityID string) ([]string, error) {
	links, err := s.client.ResourceLink.Query().
		Where(resourcelink.Or(
			resourcelink.And(resourcelink.SourceAppEQ(app), resourcelink.SourceEntityIDEQ(entityID)),
			resourcelink.And(resourcelink.TargetAppEQ(app), resourcelink.TargetEntityIDEQ(entityID)),
		)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("trace seeds by entity: %w", err)
	}
	return uniqueEventIDsFromLinks(links, func(link *gen.ResourceLink, id string) bool {
		return (link.SourceApp == app && link.SourceEntityID == entityID && id == link.SourceEventID) ||
			(link.TargetApp == app && link.TargetEntityID == entityID && id == link.TargetEventID)
	}), nil
}

func uniqueEventIDsFromLinks(links []*gen.ResourceLink, keep func(*gen.ResourceLink, string) bool) []string {
	seen := map[string]bool{}
	var seeds []string
	for _, link := range links {
		for _, id := range []string{link.SourceEventID, link.TargetEventID} {
			if id == "" || seen[id] {
				continue
			}
			if keep != nil && !keep(link, id) {
				continue
			}
			seen[id] = true
			seeds = append(seeds, id)
		}
	}
	return seeds
}

func (s *ResourceChainStore) traceNeighborLinks(ctx context.Context, eventID string, direction types.LineageTraceDirection) ([]*gen.ResourceLink, error) {
	q := s.client.ResourceLink.Query()
	switch direction {
	case types.LineageTraceDirectionDown:
		q = q.Where(resourcelink.SourceEventIDEQ(eventID))
	case types.LineageTraceDirectionUp:
		q = q.Where(resourcelink.TargetEventIDEQ(eventID))
	default:
		q = q.Where(resourcelink.Or(
			resourcelink.SourceEventIDEQ(eventID),
			resourcelink.TargetEventIDEQ(eventID),
		))
	}
	links, err := q.Order(resourcelink.ByCreatedAt()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("trace neighbors for %s: %w", eventID, err)
	}
	return links, nil
}

func rememberTraceEndpoint(dst map[string]types.LineageTraceNode, eventID, app, entityID, capability string) {
	if eventID == "" {
		return
	}
	dst[eventID] = types.LineageTraceNode{
		EventID:    eventID,
		App:        app,
		EntityID:   entityID,
		Capability: capability,
	}
}

func traceNextEventIDs(cur string, link *gen.ResourceLink, direction types.LineageTraceDirection) []string {
	switch direction {
	case types.LineageTraceDirectionDown:
		if link.SourceEventID == cur {
			return []string{link.TargetEventID}
		}
	case types.LineageTraceDirectionUp:
		if link.TargetEventID == cur {
			return []string{link.SourceEventID}
		}
	default:
		var out []string
		if link.SourceEventID == cur {
			out = append(out, link.TargetEventID)
		}
		if link.TargetEventID == cur {
			out = append(out, link.SourceEventID)
		}
		return out
	}
	return nil
}

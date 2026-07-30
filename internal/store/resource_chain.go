// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

// ---------------------------------------------------------------------------
// ResourceChainStore
// ---------------------------------------------------------------------------

// ResourceChainStore provides query methods for resource tag and lineage lookups.
type ResourceChainStore struct {
	client *gen.Client
}

// NewResourceChainStore creates a ResourceChainStore with the given ent client.
func NewResourceChainStore(client *gen.Client) *ResourceChainStore {
	return &ResourceChainStore{client: client}
}

// FindResourcesByTag returns DataEvents matching a tag key-value pair,
// ordered by created_at descending. Supports limit + opaque cursor pagination.
func (s *ResourceChainStore) FindResourcesByTag(ctx context.Context, key, value string, limit int, cursor string) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tagJSON := fmt.Sprintf(`{%q:%q}`, key, value)
	q := s.client.DataEvent.Query().
		Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP("tags @> $1", tagJSON))
		}).
		Order(dataevent.ByCreatedAt(sql.OrderDesc())).
		Limit(limit + 1)

	if cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("find resources by tag: %w", err)
	}

	result := make([]*gen.DataEvent, len(events))
	for i, e := range events {
		result[i] = &gen.DataEvent{
			EventID:    e.EventID,
			EventType:  e.EventType,
			Source:     e.Source,
			Capability: e.Capability,
			Operation:  e.Operation,
			App:        e.App,
			EntityID:   e.EntityID,
			CreatedAt:  e.CreatedAt,
		}
		if e.Data != nil {
			result[i].Data = schema.JSON(e.Data)
		}
		if e.Tags != nil {
			result[i].Tags = schema.JSON(e.Tags)
		}
	}

	var nextCursor string
	if len(result) > limit {
		nextCursor = result[limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		result = result[:limit]
	}

	return result, nextCursor, nil
}

// FindResourceLinks returns all links involving any of the given event IDs,
// either as source or target.
func (s *ResourceChainStore) FindResourceLinks(ctx context.Context, eventIDs []string) ([]*gen.ResourceLink, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}

	links, err := s.client.ResourceLink.Query().
		Where(resourcelink.Or(
			resourcelink.SourceEventIDIn(eventIDs...),
			resourcelink.TargetEventIDIn(eventIDs...),
		)).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find resource links: %w", err)
	}

	result := make([]*gen.ResourceLink, len(links))
	for i, l := range links {
		result[i] = &gen.ResourceLink{
			ID:               l.ID,
			SourceEventID:    l.SourceEventID,
			TargetEventID:    l.TargetEventID,
			SourceApp:        l.SourceApp,
			TargetApp:        l.TargetApp,
			SourceCapability: l.SourceCapability,
			TargetCapability: l.TargetCapability,
			SourceEntityID:   l.SourceEntityID,
			TargetEntityID:   l.TargetEntityID,
			PipelineRunID:    l.PipelineRunID,
			PipelineName:     l.PipelineName,
			CreatedAt:        l.CreatedAt,
		}
	}

	return result, nil
}

// FindRelations returns upstream and downstream resource references
// for a specific resource identified by appName + entity_id.
func (s *ResourceChainStore) FindRelations(ctx context.Context, appName, entityID string) (*schema.ResourceRelations, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	relations := &schema.ResourceRelations{
		App:        appName,
		EntityID:   entityID,
		Upstream:   []schema.ResourceRef{},
		Downstream: []schema.ResourceRef{},
	}

	downLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.SourceApp(appName),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find downstream: %w", err)
	}
	for _, l := range downLinks {
		relations.Downstream = append(relations.Downstream, schema.ResourceRef{
			App:          l.TargetApp,
			EntityID:     l.TargetEntityID,
			Capability:   l.TargetCapability,
			PipelineName: l.PipelineName,
		})
	}

	upLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.TargetApp(appName),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find upstream: %w", err)
	}
	for _, l := range upLinks {
		relations.Upstream = append(relations.Upstream, schema.ResourceRef{
			App:          l.SourceApp,
			EntityID:     l.SourceEntityID,
			Capability:   l.SourceCapability,
			PipelineName: l.PipelineName,
		})
	}

	return relations, nil
}

// FindNodeRelations returns upstream and downstream edges for a node identified
// by (appName, capability, entityID). Optional pipelineName filter and time window.
func (s *ResourceChainStore) FindNodeRelations(ctx context.Context, appName, capability, entityID, pipelineName string, since time.Duration) ([]schema.ResourceEdge, []schema.ResourceEdge, error) {
	if s == nil || s.client == nil {
		return nil, nil, nil
	}

	base := func() *gen.ResourceLinkQuery {
		q := s.client.ResourceLink.Query()
		if pipelineName != "" {
			q = q.Where(resourcelink.PipelineName(pipelineName))
		}
		if since > 0 {
			q = q.Where(resourcelink.CreatedAtGT(time.Now().Add(-since)))
		}
		return q
	}

	// downstream: source = this node
	downLinks, err := base().
		Where(
			resourcelink.SourceApp(appName),
			resourcelink.SourceCapability(capability),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find downstream edges: %w", err)
	}

	// upstream: target = this node
	upLinks, err := base().
		Where(
			resourcelink.TargetApp(appName),
			resourcelink.TargetCapability(capability),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find upstream edges: %w", err)
	}

	toEdges := func(links []*gen.ResourceLink) []schema.ResourceEdge {
		edges := make([]schema.ResourceEdge, len(links))
		for i, l := range links {
			edges[i] = schema.ResourceEdge{
				SourceApp:        l.SourceApp,
				SourceCapability: l.SourceCapability,
				SourceEntityID:   l.SourceEntityID,
				TargetApp:        l.TargetApp,
				TargetCapability: l.TargetCapability,
				TargetEntityID:   l.TargetEntityID,
				PipelineName:     l.PipelineName,
				CreatedAt:        l.CreatedAt,
			}
		}
		return edges
	}

	return toEdges(upLinks), toEdges(downLinks), nil
}

// SearchNodes returns distinct (app, capability, entity_id) tuples from
// resource_links where source_entity_id, target_entity_id, source_app,
// target_app, source_capability, or target_capability contains the query.
// cursor is a decimal offset into the deduplicated result stream; empty starts at 0.
func (s *ResourceChainStore) SearchNodes(ctx context.Context, query string, limit int, cursor string) ([]schema.ResourceRef, string, error) {
	if s == nil || s.client == nil || query == "" {
		return nil, "", nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return nil, "", fmt.Errorf("search nodes: invalid cursor")
		}
		offset = n
	}

	// Fetch candidate links using Ent-safe case-insensitive predicates.
	links, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.Or(
				resourcelink.SourceEntityIDContainsFold(query),
				resourcelink.TargetEntityIDContainsFold(query),
				resourcelink.SourceAppContainsFold(query),
				resourcelink.TargetAppContainsFold(query),
				resourcelink.SourceCapabilityContainsFold(query),
				resourcelink.TargetCapabilityContainsFold(query),
			),
		).
		Order(resourcelink.ByCreatedAt(sql.OrderDesc())).
		Limit((offset + limit + 1) * 2). // over-fetch to allow in-memory dedup + cursor window
		All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("search nodes: %w", err)
	}

	// Deduplicate by (app, capability, entity_id) in Go memory.
	seen := make(map[string]bool)
	var results []schema.ResourceRef
	lowerQuery := strings.ToLower(query)

	for _, rl := range links {
		addSourceResult(rl, lowerQuery, seen, &results)
		addTargetResult(rl, lowerQuery, seen, &results)
	}

	if offset > len(results) {
		return nil, "", nil
	}
	window := results[offset:]
	nextCursor := ""
	if len(window) > limit {
		window = window[:limit]
		nextCursor = strconv.Itoa(offset + limit)
	}

	return window, nextCursor, nil
}

// addSourceResult adds the source side of a resource link to results
// if any of its fields match the query.
func addSourceResult(rl *gen.ResourceLink, lowerQuery string, seen map[string]bool, results *[]schema.ResourceRef) {
	if !matchesField(rl.SourceEntityID, rl.SourceApp, rl.SourceCapability, lowerQuery) {
		return
	}
	key := rl.SourceApp + "|" + rl.SourceCapability + "|" + rl.SourceEntityID
	if seen[key] {
		return
	}
	seen[key] = true
	*results = append(*results, schema.ResourceRef{
		App:        rl.SourceApp,
		Capability: rl.SourceCapability,
		EntityID:   rl.SourceEntityID,
	})
}

// addTargetResult adds the target side of a resource link to results
// if any of its fields match the query.
func addTargetResult(rl *gen.ResourceLink, lowerQuery string, seen map[string]bool, results *[]schema.ResourceRef) {
	if !matchesField(rl.TargetEntityID, rl.TargetApp, rl.TargetCapability, lowerQuery) {
		return
	}
	key := rl.TargetApp + "|" + rl.TargetCapability + "|" + rl.TargetEntityID
	if seen[key] {
		return
	}
	seen[key] = true
	*results = append(*results, schema.ResourceRef{
		App:        rl.TargetApp,
		Capability: rl.TargetCapability,
		EntityID:   rl.TargetEntityID,
	})
}

// matchesField returns true if any of the given fields contain the query (case-insensitive).
func matchesField(entityID, appName, capability, lowerQuery string) bool {
	return strings.Contains(strings.ToLower(entityID), lowerQuery) ||
		strings.Contains(strings.ToLower(appName), lowerQuery) ||
		strings.Contains(strings.ToLower(capability), lowerQuery)
}

// ParameterIsExpired checks whether the given access token parameter has expired.
func ParameterIsExpired(p gen.Parameter) bool {
	return p.ExpiredAt.Before(time.Now())
}

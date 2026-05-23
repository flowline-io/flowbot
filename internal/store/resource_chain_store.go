package store

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/model"
)

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
func (s *ResourceChainStore) FindResourcesByTag(ctx context.Context, key, value string, limit int, cursor string) ([]*model.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tagJSON := fmt.Sprintf(`{"%s":"%s"}`, key, value)
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

	result := make([]*model.DataEvent, len(events))
	for i, e := range events {
		result[i] = &model.DataEvent{
			EventID:   e.EventID,
			EventType: e.EventType,
			Source:    e.Source,
			Capability: e.Capability,
			Operation: e.Operation,
			Backend:   e.Backend,
			App:       e.App,
			EntityID:  e.EntityID,
			CreatedAt: e.CreatedAt,
		}
		if e.Data != nil {
			result[i].Data = model.JSON(e.Data)
		}
		if e.Tags != nil {
			result[i].Tags = model.JSON(e.Tags)
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
func (s *ResourceChainStore) FindResourceLinks(ctx context.Context, eventIDs []string) ([]*model.ResourceLink, error) {
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

	result := make([]*model.ResourceLink, len(links))
	for i, l := range links {
		result[i] = &model.ResourceLink{
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
// for a specific resource identified by app + entity_id.
func (s *ResourceChainStore) FindRelations(ctx context.Context, app, entityID string) (*model.ResourceRelations, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	relations := &model.ResourceRelations{
		App:        app,
		EntityID:   entityID,
		Upstream:   []model.ResourceRef{},
		Downstream: []model.ResourceRef{},
	}

	downLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.SourceApp(app),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find downstream: %w", err)
	}
	for _, l := range downLinks {
		relations.Downstream = append(relations.Downstream, model.ResourceRef{
			App:          l.TargetApp,
			EntityID:     l.TargetEntityID,
			Capability:   l.TargetCapability,
			PipelineName: l.PipelineName,
		})
	}

	upLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.TargetApp(app),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find upstream: %w", err)
	}
	for _, l := range upLinks {
		relations.Upstream = append(relations.Upstream, model.ResourceRef{
			App:          l.SourceApp,
			EntityID:     l.SourceEntityID,
			Capability:   l.SourceCapability,
			PipelineName: l.PipelineName,
		})
	}

	return relations, nil
}

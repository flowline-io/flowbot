package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var relationsWebserviceRules = []webservice.Rule{
	webservice.Get("/relations", relationsPage),
	webservice.Get("/relations/tree", relationsTree),
	webservice.Get("/relations/graph", relationsGraph),
	webservice.Get("/relations/search", relationsSearch),
	webservice.Get("/relations/detail", relationsDetail),
}

func getResourceChainStore() *store.ResourceChainStore {
	if store.Database == nil || store.Database.GetClient() == nil {
		return nil
	}
	return store.NewResourceChainStore(store.Database.GetClient())
}

func relationsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.RelationsPage(ctx.Context(), pages.RelationsPageParams{}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsTree(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	nodeParam := ctx.Query("node")
	if nodeParam == "" {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "relations.empty.search_hint")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	parts := strings.SplitN(nodeParam, "|", 3)
	if len(parts) != 3 {
		ctx.Status(http.StatusBadRequest)
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.invalid_node_format")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	app := parts[0]
	capability := parts[1]
	entityID := parts[2]

	pipeline := ctx.Query("pipeline")
	sinceRaw := ctx.Query("since")

	var since time.Duration
	if sinceRaw != "" {
		if d, err := time.ParseDuration(sinceRaw); err == nil {
			since = d
		}
	}

	rcs := getResourceChainStore()
	if rcs == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.store_unavailable")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	upstream, downstream, err := rcs.FindNodeRelations(ctx.Context(), app, capability, entityID, pipeline, since)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.failed_load_relations")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationTree(ctx.Context(), partials.RelationTreeParams{
		Node: types.ResourceRef{
			App:        app,
			Capability: capability,
			EntityID:   entityID,
		},
		Upstream:   upstream,
		Downstream: downstream,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsGraph(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	nodeParam := ctx.Query("node")
	if nodeParam == "" {
		return ctx.JSON(types.FlowGraph{Nodes: []types.FlowGraphNode{}, Edges: []types.FlowGraphEdge{}})
	}
	parts := strings.SplitN(nodeParam, "|", 3)
	if len(parts) != 3 {
		return types.Errorf(types.ErrInvalidArgument, "invalid node format")
	}
	app := parts[0]
	capability := parts[1]
	entityID := parts[2]
	pipeline := ctx.Query("pipeline")
	sinceRaw := ctx.Query("since")
	var since time.Duration
	if sinceRaw != "" {
		if d, err := time.ParseDuration(sinceRaw); err == nil {
			since = d
		}
	}
	rcs := getResourceChainStore()
	if rcs == nil {
		return types.Errorf(types.ErrUnavailable, "store unavailable")
	}
	upstream, downstream, err := rcs.FindNodeRelations(ctx.Context(), app, capability, entityID, pipeline, since)
	if err != nil {
		return types.Errorf(types.ErrInternal, "relations graph: %v", err)
	}
	return ctx.JSON(buildRelationsFlowGraph(app, capability, entityID, upstream, downstream))
}

func buildRelationsFlowGraph(app, _, _ string, upstream, downstream []types.ResourceEdge) types.FlowGraph {
	rootID := app
	seen := map[string]bool{rootID: true}
	nodes := []types.FlowGraphNode{{ID: rootID}}
	addNode := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		nodes = append(nodes, types.FlowGraphNode{ID: id})
	}
	type edgeKey struct {
		source, target, kind, name string
	}
	agg := map[edgeKey]*types.FlowGraphEdge{}
	order := make([]edgeKey, 0)
	appendEdge := func(source, target string, e types.ResourceEdge) {
		if source == "" || target == "" {
			return
		}
		addNode(source)
		addNode(target)
		kind := strings.TrimSpace(e.RunKind)
		if kind == "" {
			kind = string(types.ResourceLinkRunKindPipeline)
		}
		name := e.PipelineName
		k := edgeKey{source: source, target: target, kind: kind, name: name}
		if existing, ok := agg[k]; ok {
			if e.Status == string(types.ResourceLinkStatusFailed) {
				existing.CountFailed++
			} else {
				existing.CountOK++
			}
			return
		}
		edge := &types.FlowGraphEdge{
			Source: source,
			Target: target,
			Via:    types.FlowGraphVia{Kind: kind, Name: name},
		}
		if e.Status == string(types.ResourceLinkStatusFailed) {
			edge.CountFailed = 1
		} else {
			edge.CountOK = 1
		}
		agg[k] = edge
		order = append(order, k)
	}
	for _, e := range upstream {
		appendEdge(e.SourceApp, rootID, e)
	}
	for _, e := range downstream {
		appendEdge(rootID, e.TargetApp, e)
	}
	edges := make([]types.FlowGraphEdge, 0, len(order))
	for _, k := range order {
		edges = append(edges, *agg[k])
	}
	return types.FlowGraph{Nodes: nodes, Edges: edges}
}

func relationsSearch(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	query := ctx.Query("q")
	if query == "" {
		ctx.Type("html")
		return nil
	}

	rcs := getResourceChainStore()
	if rcs == nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.store_unavailable")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	limit := 20
	if l, err := strconv.Atoi(ctx.Query("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	results, _, err := rcs.SearchNodes(ctx.Context(), query, limit, "")
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState(webMsg(ctx, "empty.search_failed")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationSearchResults(ctx.Context(), results).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsDetail(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	detailType := ctx.Query("type")

	ctx.Type("html")
	switch detailType {
	case "node":
		app := ctx.Query("app")
		capability := ctx.Query("capability")
		entityID := ctx.Query("entity_id")
		return partials.RelationDetail(ctx.Context(), partials.RelationDetailParams{
			Type: "node",
			Node: types.ResourceRef{
				App:        app,
				Capability: capability,
				EntityID:   entityID,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	case "edge":
		sourceApp := ctx.Query("source_app")
		sourceCap := ctx.Query("source_capability")
		sourceEntity := ctx.Query("source_entity")
		targetApp := ctx.Query("target_app")
		targetCap := ctx.Query("target_capability")
		targetEntity := ctx.Query("target_entity")
		pipeline := ctx.Query("pipeline")
		createdStr := ctx.Query("created_at")
		var createdAt time.Time
		if createdStr != "" {
			createdAt, _ = time.Parse(time.RFC3339, createdStr)
		}
		return partials.RelationDetail(ctx.Context(), partials.RelationDetailParams{
			Type: "edge",
			Edge: types.ResourceEdge{
				SourceApp:        sourceApp,
				SourceCapability: sourceCap,
				SourceEntityID:   sourceEntity,
				TargetApp:        targetApp,
				TargetCapability: targetCap,
				TargetEntityID:   targetEntity,
				PipelineName:     pipeline,
				CreatedAt:        createdAt,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	default:
		return partials.EmptyState(webMsg(ctx, "empty.invalid_detail_type")).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
}

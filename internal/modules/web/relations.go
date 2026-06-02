package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var relationsWebserviceRules = []webservice.Rule{
	webservice.Get("/relations", relationsPage, route.WithNotAuth()),
	webservice.Get("/relations/tree", relationsTree, route.WithNotAuth()),
	webservice.Get("/relations/search", relationsSearch, route.WithNotAuth()),
	webservice.Get("/relations/detail", relationsDetail, route.WithNotAuth()),
}

func getResourceChainStore() *store.ResourceChainStore {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewResourceChainStore(client)
}

func relationsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.RelationsPage(pages.RelationsPageParams{}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsTree(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	nodeParam := ctx.Query("node")
	if nodeParam == "" {
		ctx.Type("html")
		return partials.EmptyState("Search for a resource entity ID to explore relations").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	parts := strings.SplitN(nodeParam, "|", 3)
	if len(parts) != 3 {
		ctx.Status(http.StatusBadRequest)
		ctx.Type("html")
		return partials.EmptyState("Invalid node format. Use app|capability|entity_id").Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	upstream, downstream, err := rcs.FindNodeRelations(ctx.Context(), app, capability, entityID, pipeline, since)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load relations").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationTree(partials.RelationTreeParams{
		Node: schema.ResourceRef{
			App:        app,
			Capability: capability,
			EntityID:   entityID,
		},
		Upstream:   upstream,
		Downstream: downstream,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	limit := 20
	if l, err := strconv.Atoi(ctx.Query("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	results, _, err := rcs.SearchNodes(ctx.Context(), query, limit, "")
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Search failed").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationSearchResults(results).Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return partials.RelationDetail(partials.RelationDetailParams{
			Type: "node",
			Node: schema.ResourceRef{
				App:        app,
				Capability: capability,
				EntityID:   entityID,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	case "edge":
		sourceApp := ctx.Query("source_app")
		sourceEntity := ctx.Query("source_entity")
		targetApp := ctx.Query("target_app")
		targetEntity := ctx.Query("target_entity")
		pipeline := ctx.Query("pipeline")
		createdStr := ctx.Query("created_at")
		var createdAt time.Time
		if createdStr != "" {
			createdAt, _ = time.Parse(time.RFC3339, createdStr)
		}
		return partials.RelationDetail(partials.RelationDetailParams{
			Type: "edge",
			Edge: schema.ResourceEdge{
				SourceApp:      sourceApp,
				SourceEntityID: sourceEntity,
				TargetApp:      targetApp,
				TargetEntityID: targetEntity,
				PipelineName:   pipeline,
				CreatedAt:      createdAt,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	default:
		return partials.EmptyState("Invalid detail type").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
}

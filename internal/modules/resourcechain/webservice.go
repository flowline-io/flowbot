package resourcechain

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func queryOrParam(ctx fiber.Ctx, name string) string {
	v := ctx.Query(name)
	if v == "_" {
		return ""
	}
	if v != "" {
		return v
	}
	return ctx.Params(name)
}

var webserviceRules = []webservice.Rule{
	webservice.Get("/resource-chain", queryByTag),
	webservice.Get("/:app/:entity_id/relations", getRelations),
}

func queryByTag(ctx fiber.Ctx) error {
	key := ctx.Query("key")
	value := ctx.Query("value")
	if key == "" || value == "" {
		return types.Errorf(types.ErrInvalidArgument, "key and value query params are required")
	}

	limit := 20
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	cursor := ctx.Query("cursor")

	events, nextCursor, err := rcStore.FindResourcesByTag(context.Background(), key, value, limit, cursor)
	if err != nil {
		return err
	}

	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	links, err := rcStore.FindResourceLinks(context.Background(), eventIDs)
	if err != nil {
		flog.Error(fmt.Errorf("resourcechain: find links: %w", err))
	}

	type resEntry struct {
		EntityID   string `json:"entity_id"`
		App        string `json:"app"`
		Capability string `json:"capability"`
		EventID    string `json:"event_id"`
		CreatedAt  string `json:"created_at"`
	}
	type linkEntry struct {
		Source       resEntry `json:"source"`
		Target       resEntry `json:"target"`
		PipelineName string   `json:"pipeline_name"`
		CreatedAt    string   `json:"created_at"`
	}

	resources := make([]resEntry, len(events))
	for i, e := range events {
		resources[i] = resEntry{
			EntityID: e.EntityID, App: e.App, Capability: e.Capability,
			EventID: e.EventID, CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	linkEntries := make([]linkEntry, 0, len(links))
	for _, l := range links {
		linkEntries = append(linkEntries, linkEntry{
			Source:       resEntry{EntityID: l.SourceEntityID, App: l.SourceApp},
			Target:       resEntry{EntityID: l.TargetEntityID, App: l.TargetApp},
			PipelineName: l.PipelineName,
			CreatedAt:    l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	result := types.KV{
		"tag":       types.KV{"key": key, "value": value},
		"resources": resources,
		"links":     linkEntries,
	}
	if nextCursor != "" {
		result["cursor"] = nextCursor
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func getRelations(ctx fiber.Ctx) error {
	app := queryOrParam(ctx, "app")
	entityID := queryOrParam(ctx, "entity_id")
	if app == "" || entityID == "" {
		return types.Errorf(types.ErrInvalidArgument, "app and entity_id path params are required")
	}

	relations, err := rcStore.FindRelations(context.Background(), app, entityID)
	if err != nil {
		return err
	}
	if relations == nil {
		relations = &model.ResourceRelations{
			App: app, EntityID: entityID,
			Upstream: []model.ResourceRef{}, Downstream: []model.ResourceRef{},
		}
	}

	return ctx.JSON(protocol.NewSuccessResponse(relations))
}

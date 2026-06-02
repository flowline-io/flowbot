package web

import (
	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

var eventWebserviceRules = []webservice.Rule{
	webservice.Get("/events", eventsPage, route.WithNotAuth()),
	webservice.Get("/events/data-events", dataEventsTable, route.WithNotAuth()),
	webservice.Get("/events/webhook-logs", webhookLogsTable, route.WithNotAuth()),
	webservice.Get("/events/payload/:eventID", eventPayload, route.WithNotAuth()),
}

func requireAdmin(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scopes := route.GetScopes(ctx)
	if !auth.HasScope(scopes, auth.ScopeAdmin) {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString("Admin access required")
	}
	return nil
}

func getEventStore() *store.EventStore {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewEventStore(client)
}

func hasWebhookData(e *gen.DataEvent) bool {
	if e.Data == nil {
		return false
	}
	_, hasMethod := e.Data["_webhook_method"]
	return hasMethod
}

func eventsPage(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	s := getEventStore()
	var events []*gen.DataEvent
	var nextCursor string
	if s != nil {
		events, nextCursor, _ = s.ListDataEvents(ctx.Context(), store.ListDataEventsOptions{
			Limit: 20,
		})
	}

	ctx.Type("html")
	return pages.EventsPage(pages.EventsPageParams{
		ActiveTab:  "data-events",
		Sources:    sources,
		EventTypes: eventTypes,
		Events:     events,
		NextCursor: nextCursor,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func dataEventsTable(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sourceFilter := ctx.Query("source")
	typeFilter := ctx.Query("type")
	cursor := ctx.Query("cursor")

	s := getEventStore()
	if s == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	events, nextCursor, err := s.ListDataEvents(ctx.Context(), store.ListDataEventsOptions{
		Limit:     20,
		Cursor:    cursor,
		Source:    sourceFilter,
		EventType: typeFilter,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load events").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	ctx.Type("html")
	return partials.DataEventsTable(sources, eventTypes, sourceFilter, typeFilter, events, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func webhookLogsTable(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sourceFilter := ctx.Query("source")
	typeFilter := ctx.Query("type")
	cursor := ctx.Query("cursor")

	s := getEventStore()
	if s == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	events, nextCursor, err := s.ListDataEvents(ctx.Context(), store.ListDataEventsOptions{
		Limit:     20,
		Cursor:    cursor,
		Source:    sourceFilter,
		EventType: typeFilter,
		Webhook:   true,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load webhook logs").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	ctx.Type("html")
	return partials.WebhookLogsTable(sources, eventTypes, sourceFilter, typeFilter, events, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func eventPayload(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	eventID := ctx.Params("eventID")

	s := getEventStore()
	if s == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	found, err := s.GetDataEventByEventID(ctx.Context(), eventID)
	if err != nil || found == nil {
		ctx.Type("html")
		return partials.EmptyState("Event not found").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	payloadJSON := "{}"
	if found.Data != nil {
		if b, err := sonic.Marshal(found.Data); err == nil {
			payloadJSON = string(b)
		}
	}

	if hasWebhookData(found) {
		headersJSON := "{}"
		bodyJSON := ""
		bodyTruncated := false
		if found.Data != nil {
			if h, ok := found.Data["_webhook_headers"]; ok {
				if b, err := sonic.Marshal(h); err == nil {
					headersJSON = string(b)
				}
			}
			if b, ok := found.Data["_webhook_body"]; ok {
				if s, ok := b.(string); ok {
					bodyJSON = s
				}
			}
			if t, ok := found.Data["_webhook_body_truncated"]; ok {
				if v, ok := t.(bool); ok {
					bodyTruncated = v
				}
			}
		}
		ctx.Type("html")
		return partials.WebhookPayloadDetail(headersJSON, bodyJSON, bodyTruncated).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.EventPayloadDetail(payloadJSON).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

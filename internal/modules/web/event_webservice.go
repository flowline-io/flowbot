package web

import (
	"fmt"
	"strconv"
	"time"

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
	webservice.Get("/events/filtered-events", filteredEventsTable, route.WithNotAuth()),
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

// parseTimeParam parses a time query parameter supporting RFC3339 and datetime-local formats.
func parseTimeParam(s string) (time.Time, error) {
	formats := []string{time.RFC3339, "2006-01-02T15:04"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}

// parseEventFilterParams extracts filter parameters from the request query string.
func parseEventFilterParams(c fiber.Ctx) store.ListDataEventsOptions {
	opts := store.ListDataEventsOptions{
		Source:    c.Query("source"),
		EventType: c.Query("type"),
		Search:    c.Query("search"),
	}

	if p := c.Query("pipeline"); p != "" {
		opts.PipelineName = p
	}

	parsePagination(c, &opts)
	parseTimeRange(c, &opts)

	if c.Query("tab") == "webhook-logs" {
		opts.Webhook = true
	}

	return opts
}

// parsePagination extracts per_page and page parameters into the options.
func parsePagination(c fiber.Ctx, opts *store.ListDataEventsOptions) {
	perPage := 20
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			perPage = v
		}
	}
	opts.Limit = perPage

	page := 1
	if pg := c.Query("page"); pg != "" {
		if v, err := strconv.Atoi(pg); err == nil && v > 0 {
			page = v
		}
	}
	opts.Offset = (page - 1) * perPage
}

// parseTimeRange extracts time_start and time_end parameters into the options.
// If the end time is before the start time, both are discarded.
func parseTimeRange(c fiber.Ctx, opts *store.ListDataEventsOptions) {
	if ts := c.Query("time_start"); ts != "" {
		if t, err := parseTimeParam(ts); err == nil {
			opts.TimeStart = &t
		}
	}
	if te := c.Query("time_end"); te != "" {
		if t, err := parseTimeParam(te); err == nil {
			opts.TimeEnd = &t
		}
	}

	if opts.TimeStart != nil && opts.TimeEnd != nil && opts.TimeEnd.Before(*opts.TimeStart) {
		opts.TimeStart = nil
		opts.TimeEnd = nil
	}
}

func eventsPage(ctx fiber.Ctx) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	s := getEventStore()
	var pipelineNames []string
	if s != nil {
		pipelineNames, _ = s.ListDistinctEventPipelineNames(ctx.Context())
	}

	ctx.Type("html")
	return pages.EventsPage(pages.EventsPageParams{
		Sources:       sources,
		EventTypes:    eventTypes,
		PipelineNames: pipelineNames,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func filteredEventsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	s := getEventStore()
	if s == nil {
		c.Type("html")
		return partials.EmptyState("Store not available").Render(c.Context(), c.Response().BodyWriter())
	}

	opts := parseEventFilterParams(c)

	total, err := s.CountDataEvents(c.Context(), opts)
	if err != nil {
		c.Type("html")
		return partials.EmptyState("Failed to count events").Render(c.Context(), c.Response().BodyWriter())
	}

	events, _, err := s.ListDataEvents(c.Context(), opts)
	if err != nil {
		c.Type("html")
		return partials.EmptyState("Failed to load events").Render(c.Context(), c.Response().BodyWriter())
	}

	// Build event ID list for pipeline name lookups
	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	runMap, _ := s.GetPipelineRunsForEvents(c.Context(), eventIDs)

	sources := types.EventFilterCache.Sources()
	eventTypes := types.EventFilterCache.EventTypes()

	perPage := opts.Limit
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	currentPage := 1
	if pg := c.Query("page"); pg != "" {
		if v, err := strconv.Atoi(pg); err == nil && v > 0 {
			currentPage = v
		}
	}
	if currentPage > totalPages && totalPages > 0 {
		currentPage = totalPages
	}

	pageInfo := partials.PageInfo{
		Page:       currentPage,
		TotalPages: totalPages,
		Total:      total,
		PerPage:    perPage,
		HasPrev:    currentPage > 1,
		HasNext:    currentPage < totalPages,
	}

	c.Type("html")
	if opts.Webhook {
		return partials.WebhookLogsTable(sources, eventTypes, events, pageInfo, runMap).
			Render(c.Context(), c.Response().BodyWriter())
	}
	return partials.DataEventsTable(sources, eventTypes, events, pageInfo, runMap).
		Render(c.Context(), c.Response().BodyWriter())
}

func dataEventsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	source := c.Query("source")
	typeFilter := c.Query("type")
	cursor := c.Query("cursor")
	u := "/service/web/events/filtered-events?tab=data-events"
	if source != "" {
		u += "&source=" + source
	}
	if typeFilter != "" {
		u += "&type=" + typeFilter
	}
	if cursor != "" {
		u += "&cursor=" + cursor
	}
	c.Set("HX-Redirect", u)
	return c.SendStatus(200)
}

func webhookLogsTable(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	source := c.Query("source")
	typeFilter := c.Query("type")
	cursor := c.Query("cursor")
	u := "/service/web/events/filtered-events?tab=webhook-logs"
	if source != "" {
		u += "&source=" + source
	}
	if typeFilter != "" {
		u += "&type=" + typeFilter
	}
	if cursor != "" {
		u += "&cursor=" + cursor
	}
	c.Set("HX-Redirect", u)
	return c.SendStatus(200)
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

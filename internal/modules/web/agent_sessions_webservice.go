package web

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var agentSessionsWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-sessions", agentSessionsPage, route.WithNotAuth()),
	webservice.Get("/agent-sessions/list", agentSessionsTable, route.WithNotAuth()),
	webservice.Get("/agent-sessions/:id", agentSessionDetailPage, route.WithNotAuth()),
	webservice.Get("/agent-sessions/:id/entries/:entryID/payload", agentSessionEntryPayload, route.WithNotAuth()),
	webservice.Get("/agent-sessions/:id/events", agentSessionEvents, route.WithNotAuth()),
	webservice.Post("/agent-sessions/:id/confirm", agentSessionConfirm, route.WithNotAuth()),
}

func agentSessionsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, nextCursor, err := listAgentSessionModels(ctx, "")
	if err != nil {
		return types.Errorf(types.ErrInternal, "list agent sessions: %v", err)
	}
	ctx.Type("html")
	return pages.AgentSessionsPage(items, nextCursor).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSessionsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	cursor := ctx.Query("cursor")
	items, nextCursor, err := listAgentSessionModels(ctx, cursor)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load agent sessions")
	}
	ctx.Type("html")
	return partials.AgentSessionTable(items, nextCursor).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSessionDetailPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	sessionID := ctx.Params("id")
	if sessionID == "" {
		return ctx.Status(http.StatusBadRequest).SendString("session id required")
	}

	row, err := store.Database.GetChatSession(ctx.Context(), sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("session not found")
		}
		return types.Errorf(types.ErrInternal, "get chat session: %v", err)
	}

	entries, err := store.Database.ListChatSessionEntries(ctx.Context(), sessionID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list chat session entries: %v", err)
	}

	ctx.Type("html")
	return pages.AgentSessionDetailPage(
		mapAgentSession(row),
		mapAgentSessionEntries(entries),
	).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentSessionEntryPayload(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	sessionID := ctx.Params("id")
	entryID := ctx.Params("entryID")
	if sessionID == "" || entryID == "" {
		ctx.Type("html")
		return partials.EmptyState("Invalid session or entry id").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	entry, err := store.Database.GetChatSessionEntryInSession(ctx.Context(), sessionID, entryID)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Entry not found").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.AgentSessionEntryPayload(formatEntryPayloadForDisplay(entry.Payload)).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func listAgentSessionModels(ctx fiber.Ctx, cursor string) ([]model.AgentSession, string, error) {
	if store.Database == nil {
		return nil, "", errors.New("store not available")
	}
	rows, nextCursor, err := store.Database.ListChatSessions(ctx.Context(), store.ListChatSessionsOptions{
		Limit:  20,
		Cursor: cursor,
	})
	if err != nil {
		return nil, "", err
	}
	items := make([]model.AgentSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAgentSession(row))
	}
	return items, nextCursor, nil
}

func mapAgentSession(row *gen.ChatSession) model.AgentSession {
	if row == nil {
		return model.AgentSession{}
	}
	return model.AgentSession{
		Flag:      row.Flag,
		Title:     row.Title,
		UID:       row.UID,
		LeafID:    row.LeafID,
		State:     chatSessionStateLabel(row.State),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapAgentSessionEntries(rows []*gen.ChatSessionEntry) []model.AgentSessionEntry {
	items := make([]model.AgentSessionEntry, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, model.AgentSessionEntry{
			Flag:        row.Flag,
			SessionID:   row.SessionID,
			ParentID:    row.ParentID,
			EntryType:   row.EntryType,
			PayloadJSON: formatEntryPayloadForDisplay(row.Payload),
			CreatedAt:   row.CreatedAt,
		})
	}
	return items
}

func chatSessionStateLabel(state int) string {
	switch schema.ChatSessionState(state) {
	case schema.ChatSessionActive:
		return "Active"
	case schema.ChatSessionClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

func formatEntryPayloadForDisplay(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	return partials.FormatEntryPayload(payload)
}

func agentSessionConfirm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	sessionID := ctx.Params("id")
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("session not found")
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).SendString("forbidden")
		}
		return types.Errorf(types.ErrInternal, "confirm: %v", err)
	}
	var body struct {
		ID       string `json:"id"`
		Approved bool   `json:"approved"`
		Mode     string `json:"mode"`
		Pattern  string `json:"pattern"`
	}
	if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	reason := chatagent.ConfirmReasonDenied
	if body.Approved {
		reason = chatagent.ConfirmReasonApproved
	}
	mode := chatagent.ConfirmMode(body.Mode)
	if mode == "" {
		if body.Approved {
			mode = chatagent.ConfirmModeOnce
		} else {
			mode = chatagent.ConfirmModeReject
		}
	}
	ok, err := chatagent.ResolveConfirm(sessionID, body.ID, body.Approved, mode, body.Pattern, reason)
	if errors.Is(err, chatagent.ErrConfirmNotFound) {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, chatagent.ErrConfirmResolved) {
		return ctx.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return types.Errorf(types.ErrInternal, "confirm: %v", err)
	}
	if !ok {
		return ctx.Status(http.StatusConflict).JSON(fiber.Map{"error": "confirm not applied"})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func agentSessionEvents(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	sessionID := ctx.Params("id")
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("session not found")
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).SendString("forbidden")
		}
		return types.Errorf(types.ErrInternal, "events: %v", err)
	}

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	// Resolve the request context before streaming. The SendStreamWriter
	// callback runs in a separate goroutine after this handler returns, when
	// Fiber has released and reused the fiber.Ctx; calling ctx.Context() from
	// inside the callback races with that release.
	reqCtx := ctx.Context()
	return ctx.SendStreamWriter(func(w *bufio.Writer) {
		hub := chatagent.GetSessionEventHub(sessionID)
		subID := fmt.Sprintf("web-%p", w)
		publisher := hub.Subscribe(subID, 32)
		defer hub.Unsubscribe(subID)

		for {
			select {
			case <-reqCtx.Done():
				return
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				switch ev.Type {
				case chatagent.EventTypeConfirm, chatagent.EventTypeConfirmResolved, chatagent.EventTypeCanceled:
					frame, err := chatagent.FormatSSEData(ev)
					if err != nil {
						return
					}
					if _, err := fmt.Fprint(w, frame); err != nil {
						return
					}
					if err := w.Flush(); err != nil {
						return
					}
				}
			}
		}
	})
}

package web

import (
	"bufio"
	"errors"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var (
	agentsWebserviceRules = []webservice.Rule{
		webservice.Get("/agents", agentsPage, route.WithNotAuth()),
		webservice.Get("/agents/list", agentsTable, route.WithNotAuth()),
		webservice.Post("/agents", agentsCreate, route.WithNotAuth()),
		// Static paths must be registered before /agents/:id.
		webservice.Post("/agents/render-markdown", agentRenderMarkdown, route.WithNotAuth()),
		webservice.Get("/agents/:id", agentChatPage, route.WithNotAuth()),
		webservice.Delete("/agents/:id", agentChatClose, route.WithNotAuth()),
		webservice.Post("/agents/:id/messages", agentChatSendMessage, route.WithNotAuth()),
		webservice.Post("/agents/:id/cancel", agentChatCancel, route.WithNotAuth()),
		webservice.Post("/agents/:id/confirm", agentChatConfirm, route.WithNotAuth()),
		webservice.Get("/agents/:id/events", agentChatEvents, route.WithNotAuth()),
		webservice.Get("/agents/:id/context", agentChatContext, route.WithNotAuth()),
	}

	webChatAgentService = chatagent.NewService()
)

func agentsEndpoints() partials.ChatAgentEndpoints {
	return partials.ChatAgentEndpoints{
		CreateURL:         "/service/web/agents",
		ListURL:           "/service/web/agents/list",
		DetailURLTemplate: "/service/web/agents/{id}",
		RenderMarkdownURL: "/service/web/agents/render-markdown",
	}
}

func agentChatEndpoints(sessionID string) partials.ChatAgentEndpoints {
	prefix := "/service/web/agents/" + sessionID
	return partials.ChatAgentEndpoints{
		DetailURLTemplate: "/service/web/agents/{id}",
		MessagesURL:       prefix + "/messages",
		CancelURL:         prefix + "/cancel",
		CloseURL:          prefix,
		ConfirmURL:        prefix + "/confirm",
		EventsURL:         prefix + "/events",
		InspectURL:        "/service/web/agent-sessions/" + sessionID,
		RenderMarkdownURL: "/service/web/agents/render-markdown",
		ContextURL:        prefix + "/context",
	}
}

func webRequireChatAgentEnabled() error {
	if !pkgconfig.ChatAgentEnabled() {
		return types.ErrUnavailable
	}
	return nil
}

func agentsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	enabled := pkgconfig.ChatAgentEnabled()
	var items []model.AgentSession
	var nextCursor string
	if enabled {
		var err error
		items, nextCursor, err = listUserAgentSessionModels(ctx, "")
		if err != nil {
			return types.Errorf(types.ErrInternal, "list agents: %v", err)
		}
	}
	ctx.Type("html")
	return pages.AgentsPage(items, nextCursor, agentsEndpoints(), enabled).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return renderError(ctx, "Chat agent is not enabled")
	}
	cursor := ctx.Query("cursor")
	items, nextCursor, err := listUserAgentSessionModels(ctx, cursor)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load sessions")
	}
	ctx.Type("html")
	endpoints := agentsEndpoints()
	if cursor != "" {
		return partials.ChatAgentSessionListAppend(items, nextCursor, endpoints).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	return partials.ChatAgentSessionList(items, nextCursor, endpoints).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentsCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	uid, err := webUID(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	sessionID := types.Id()
	if err := chatagent.CreateSession(ctx.Context(), uid, sessionID); err != nil {
		return types.Errorf(types.ErrInternal, "create session: %v", err)
	}
	return ctx.Status(http.StatusCreated).JSON(fiber.Map{"session_id": sessionID})
}

func agentChatPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).SendString("chat agent is not enabled")
	}
	sessionID := strings.Clone(ctx.Params("id"))
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
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).SendString("forbidden")
		}
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("session not found")
		}
		return types.Errorf(types.ErrInternal, "agent chat: %v", err)
	}
	messages, err := chatagent.ListSessionMessages(ctx.Context(), sessionID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list messages: %v", err)
	}
	ctx.Type("html")
	return pages.AgentChatPage(
		mapAgentSession(row),
		mapChatMessages(messages),
		agentChatEndpoints(sessionID),
	).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func agentChatSendMessage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	// Clone before SendStreamWriter: Fiber recycles fasthttp buffers after the
	// handler returns, and concurrent /agents/render-markdown traffic can
	// overwrite Params("id") in place (e.g. Cev…7Hi4BPL -> render-markdown7Hi4BPL).
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return types.Errorf(types.ErrInternal, "send message: %v", err)
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	text := strings.Clone(strings.TrimSpace(body.Text))
	if text == "" {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty message"})
	}
	if _, ok := chatagent.GetAPIRunState(sessionID); ok {
		return ctx.Status(http.StatusConflict).JSON(fiber.Map{"error": chatagent.ErrRunInFlight.Error()})
	}

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	baseCtx := ctx.Context()
	return ctx.SendStreamWriter(func(w *bufio.Writer) {
		sse := &chatagent.BufioSSEWriter{W: w}
		chatagent.StreamAPIRun(baseCtx, webChatAgentService, sessionID, text, sse)
	})
}

func agentChatClose(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return types.Errorf(types.ErrInternal, "close session: %v", err)
	}
	if err := chatagent.CloseSession(ctx.Context(), sessionID); err != nil {
		return types.Errorf(types.ErrInternal, "close session: %v", err)
	}
	chatagent.ClearAPIRunState(sessionID, nil)
	return ctx.SendStatus(fiber.StatusNoContent)
}

func agentChatCancel(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return types.Errorf(types.ErrInternal, "cancel: %v", err)
	}
	chatagent.CancelSessionRun(sessionID)
	if state, ok := chatagent.GetAPIRunState(sessionID); ok {
		if pub := state.Publisher(); pub != nil {
			_ = pub.Publish(chatagent.StreamEvent{
				Type:    chatagent.EventTypeCanceled,
				Message: "run canceled by user",
			})
		}
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func agentChatConfirm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
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

func agentRenderMarkdown(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	if strings.TrimSpace(body.Text) == "" {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty text"})
	}
	return ctx.JSON(fiber.Map{"html": partials.RenderChatAgentMarkdownHTML(body.Text)})
}

func agentChatContext(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "chat agent is not enabled"})
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return types.Errorf(types.ErrInternal, "context usage: %v", err)
	}
	report, err := chatagent.BuildContextUsageReport(ctx.Context(), sessionID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(report)
}

func agentChatEvents(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return ctx.Status(http.StatusServiceUnavailable).SendString("chat agent is not enabled")
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return ctx.Status(http.StatusNotFound).SendString("session not found")
		}
		if errors.Is(err, types.ErrForbidden) {
			return ctx.Status(http.StatusForbidden).SendString("forbidden")
		}
		return types.Errorf(types.ErrInternal, "events: %v", err)
	}
	return streamWebSessionEvents(ctx, sessionID)
}

func listUserAgentSessionModels(ctx fiber.Ctx, cursor string) ([]model.AgentSession, string, error) {
	if store.Database == nil {
		return nil, "", errors.New("store not available")
	}
	uid := getUID(ctx)
	active := int(schema.ChatSessionActive)
	rows, nextCursor, err := store.Database.ListChatSessions(ctx.Context(), store.ListChatSessionsOptions{
		Limit:  20,
		Cursor: cursor,
		UID:    uid,
		State:  &active,
	})
	if err != nil {
		return nil, "", err
	}
	leafBySession := make(map[string]string, len(rows))
	items := make([]model.AgentSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAgentSession(row))
		leafBySession[row.Flag] = row.LeafID
	}
	if len(leafBySession) > 0 {
		if durations, err := chatagent.SumSessionsRunDurationMs(ctx.Context(), leafBySession); err == nil {
			for i := range items {
				items[i].TotalDurationMs = durations[items[i].Flag]
			}
		}
	}
	return items, nextCursor, nil
}

func mapChatMessages(messages []chatagent.HistoryMessage) []model.AgentChatMessage {
	usePersisted := chatagent.HasPersistedToolResults(messages)
	out := make([]model.AgentChatMessage, 0, len(messages))
	for _, m := range messages {
		if !usePersisted && m.Role == "assistant" && (m.Kind == "" || m.Kind == "assistant") {
			out = append(out, partials.ClassifyHistoryMessage(m.Role, m.Text, m.CreatedAt)...)
			continue
		}
		out = append(out, mapHistoryMessage(m))
	}
	return out
}

func mapHistoryMessage(m chatagent.HistoryMessage) model.AgentChatMessage {
	kind := m.Kind
	if kind == "" {
		kind = m.Role
	}
	switch kind {
	case "tool":
		return model.AgentChatMessage{
			Role:       "tool",
			Kind:       "tool",
			ToolName:   m.ToolName,
			ToolStatus: m.ToolStatus,
			ToolStdout: m.Text,
			DurationMs: m.DurationMs,
			CreatedAt:  m.CreatedAt,
		}
	case "thinking":
		text := m.ThinkingText
		if text == "" {
			text = m.Text
		}
		return model.AgentChatMessage{
			Role:               "assistant",
			Kind:               "thinking",
			Text:               text,
			HTML:               partials.RenderChatAgentMarkdownHTML(text),
			ThinkingDurationMs: m.ThinkingDurationMs,
			CreatedAt:          m.CreatedAt,
		}
	case "user":
		return model.AgentChatMessage{
			Role:      "user",
			Kind:      "user",
			Text:      m.Text,
			HTML:      partials.FormatChatAgentMessageHTML("user", m.Text),
			CreatedAt: m.CreatedAt,
		}
	default:
		return model.AgentChatMessage{
			Role:           "assistant",
			Kind:           "assistant",
			Text:           m.Text,
			HTML:           partials.FormatChatAgentMessageHTML("assistant", m.Text),
			TurnDurationMs: m.TurnDurationMs,
			RunDurationMs:  m.RunDurationMs,
			CreatedAt:      m.CreatedAt,
		}
	}
}

package web

import (
	"bufio"
	"bytes"
	"errors"
	"io"
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
		webservice.Post("/agents/:id/pin", agentChatPin, route.WithNotAuth()),
		webservice.Delete("/agents/:id/pin", agentChatUnpin, route.WithNotAuth()),
		webservice.Post("/agents/:id/archive", agentChatArchive, route.WithNotAuth()),
		webservice.Delete("/agents/:id/archive", agentChatUnarchive, route.WithNotAuth()),
		webservice.Get("/agents/:id/settings", agentChatGetSettings, route.WithNotAuth()),
		webservice.Put("/agents/:id/settings", agentChatPutSettings, route.WithNotAuth()),
		webservice.Post("/agents/:id/messages", agentChatSendMessage, route.WithNotAuth()),
		webservice.Post("/agents/:id/media", agentChatUploadMedia, route.WithNotAuth()),
		webservice.Post("/agents/:id/cancel", agentChatCancel, route.WithNotAuth()),
		webservice.Post("/agents/:id/confirm", agentChatConfirm, route.WithNotAuth()),
		webservice.Get("/agents/:id/events", agentChatEvents, route.WithNotAuth()),
		webservice.Get("/agents/:id/context", agentChatContext, route.WithNotAuth()),
		webservice.Get("/agents/:id/todos", agentChatTodos, route.WithNotAuth()),
	}

	webChatAgentService = chatagent.NewService()
)

const (
	agentsListFilterAll           = ""
	agentsListFilterRunning       = "running"
	agentsListFilterNeedsApproval = "needs_approval"
	agentsListFilterArchived      = "archived"
)

func agentsEndpoints() partials.ChatAgentEndpoints {
	return agentsEndpointsWithFilter("")
}

func agentsEndpointsWithFilter(filter string) partials.ChatAgentEndpoints {
	return partials.ChatAgentEndpoints{
		CreateURL:            "/service/web/agents",
		ListURL:              "/service/web/agents/list",
		DetailURLTemplate:    "/service/web/agents/{id}",
		PinURLTemplate:       "/service/web/agents/{id}/pin",
		ArchiveURLTemplate:   "/service/web/agents/{id}/archive",
		Filter:               normalizeAgentsListFilter(filter),
		PendingApprovalCount: chatagent.CountPendingApprovalSessions(),
		RenderMarkdownURL:    "/service/web/agents/render-markdown",
		SelectableModels:     selectableModelOptions(),
		DefaultModel:         pkgconfig.ChatAgentChatModel(),
	}
}

func normalizeAgentsListFilter(filter string) string {
	switch strings.TrimSpace(filter) {
	case agentsListFilterRunning, agentsListFilterNeedsApproval, agentsListFilterArchived:
		return filter
	default:
		return agentsListFilterAll
	}
}

func agentChatEndpoints(sessionID string) partials.ChatAgentEndpoints {
	prefix := "/service/web/agents/" + sessionID
	return partials.ChatAgentEndpoints{
		DetailURLTemplate: "/service/web/agents/{id}",
		SettingsURL:       prefix + "/settings",
		MessagesURL:       prefix + "/messages",
		MediaURL:          prefix + "/media",
		CancelURL:         prefix + "/cancel",
		CloseURL:          prefix,
		ConfirmURL:        prefix + "/confirm",
		EventsURL:         prefix + "/events",
		InspectURL:        "/service/web/agent-sessions/" + sessionID,
		RenderMarkdownURL: "/service/web/agents/render-markdown",
		ContextURL:        prefix + "/context",
		TodosURL:          prefix + "/todos",
		SelectableModels:  selectableModelOptions(),
		DefaultModel:      pkgconfig.ChatAgentChatModel(),
	}
}

func selectableModelOptions() []partials.SelectableModelOption {
	models := chatagent.BuildSelectableModels()
	opts := make([]partials.SelectableModelOption, len(models))
	for i, m := range models {
		opts[i] = partials.SelectableModelOption{ID: m.ID, Name: m.Name}
	}
	return opts
}

func chatAgentSettingsJSONError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, types.ErrUnavailable):
		return ctx.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, types.ErrNotFound):
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	default:
		return types.Errorf(types.ErrInternal, "session settings: %v", err)
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
	filter := normalizeAgentsListFilter(ctx.Query("filter"))
	var items []model.AgentSession
	var nextCursor string
	if enabled {
		var err error
		items, nextCursor, err = listUserAgentSessionModels(ctx, "", filter)
		if err != nil {
			return types.Errorf(types.ErrInternal, "list agents: %v", err)
		}
	}
	ctx.Type("html")
	return pages.AgentsPage(items, nextCursor, agentsEndpointsWithFilter(filter), enabled).
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
	filter := normalizeAgentsListFilter(ctx.Query("filter"))
	items, nextCursor, err := listUserAgentSessionModels(ctx, cursor, filter)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load sessions")
	}
	ctx.Type("html")
	endpoints := agentsEndpointsWithFilter(filter)
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
	var body struct {
		Model         string `json:"model"`
		ThinkingLevel string `json:"thinking_level"`
	}
	if len(ctx.Body()) > 0 {
		if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}
	}
	sessionID := types.Id()
	if err := chatagent.CreateSession(ctx.Context(), uid, sessionID); err != nil {
		return types.Errorf(types.ErrInternal, "create session: %v", err)
	}
	if body.Model != "" || body.ThinkingLevel != "" {
		s := chatagent.SessionSettings{Model: body.Model, ThinkingLevel: body.ThinkingLevel}
		if err := chatagent.SetSessionSettings(ctx.Context(), sessionID, s); err != nil {
			return chatAgentSettingsJSONError(ctx, err)
		}
	}
	return ctx.Status(http.StatusCreated).JSON(fiber.Map{"session_id": sessionID})
}

func agentChatGetSettings(ctx fiber.Ctx) error {
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
		return types.Errorf(types.ErrInternal, "get settings: %v", err)
	}
	settings, err := chatagent.GetSessionSettings(ctx.Context(), sessionID)
	if err != nil {
		return chatAgentSettingsJSONError(ctx, err)
	}
	return ctx.JSON(settings)
}

func agentChatPutSettings(ctx fiber.Ctx) error {
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
		return types.Errorf(types.ErrInternal, "put settings: %v", err)
	}
	var body chatagent.SessionSettings
	if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	if err := chatagent.SetSessionSettings(ctx.Context(), sessionID, body); err != nil {
		return chatAgentSettingsJSONError(ctx, err)
	}
	chatagent.EvictHarnessPool(sessionID)
	settings, err := chatagent.GetSessionSettings(ctx.Context(), sessionID)
	if err != nil {
		return chatAgentSettingsJSONError(ctx, err)
	}
	return ctx.JSON(settings)
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
	todos, err := chatagent.ListTodoModels(ctx.Context(), sessionID)
	if err != nil {
		return types.Errorf(types.ErrInternal, "list todos: %v", err)
	}
	ctx.Type("html")
	return pages.AgentChatPage(
		mapAgentSession(row),
		mapChatMessages(messages),
		todos,
		agentChatEndpoints(sessionID),
		pendingConfirmForSession(sessionID),
	).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func pendingConfirmForSession(sessionID string) *partials.ChatAgentPendingConfirm {
	ev, ok := chatagent.LookupPendingConfirm(sessionID)
	if !ok {
		return nil
	}
	return partials.ChatAgentPendingConfirmFromEvent(partials.ChatAgentPendingConfirm{
		ID:               ev.ID,
		Tool:             ev.Tool,
		Summary:          ev.Summary,
		Permission:       ev.Permission,
		Pattern:          ev.Pattern,
		SuggestedPattern: ev.SuggestedPattern,
		SuggestAlways:    ev.SuggestAlways,
	})
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
		Text        string                    `json:"text"`
		Attachments []chatagent.AttachmentRef `json:"attachments"`
	}
	if err := sonic.Unmarshal(ctx.Body(), &body); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}
	text := strings.Clone(strings.TrimSpace(body.Text))
	attachments := append([]chatagent.AttachmentRef(nil), body.Attachments...)
	if text == "" && len(attachments) == 0 {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty message"})
	}
	if _, ok := chatagent.GetAPIRunState(sessionID); ok {
		return ctx.Status(http.StatusConflict).JSON(fiber.Map{"error": chatagent.ErrRunInFlight.Error()})
	}

	ownerUID := getUID(ctx)

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	baseCtx := ctx.Context()
	return ctx.SendStreamWriter(func(w *bufio.Writer) {
		sse := &chatagent.BufioSSEWriter{W: w}
		chatagent.StreamAPIRun(baseCtx, webChatAgentService, sessionID, text, attachments, ownerUID, sse)
	})
}

func agentChatUploadMedia(ctx fiber.Ctx) error {
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
		return types.Errorf(types.ErrInternal, "upload media: %v", err)
	}
	if err := chatagent.EnsureMediaPublicConfig(); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	file, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "open file failed"})
	}
	defer func() { _ = file.Close() }()
	var seeker io.ReadSeeker
	if rs, ok := file.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "read file failed"})
		}
		seeker = bytes.NewReader(data)
	}
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	result, err := chatagent.UploadSessionMedia(ctx.Context(), sessionID, getUID(ctx), fileHeader.Filename, mimeType, seeker, fileHeader.Size)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(result)
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

func agentChatTodos(ctx fiber.Ctx) error {
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
		return types.Errorf(types.ErrInternal, "list todos: %v", err)
	}
	todos, err := chatagent.ListTodoItems(ctx.Context(), sessionID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(fiber.Map{"todos": todos})
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

func listUserAgentSessionModels(ctx fiber.Ctx, cursor, filter string) ([]model.AgentSession, string, error) {
	if store.Database == nil {
		return nil, "", errors.New("store not available")
	}
	uid := getUID(ctx)
	filter = normalizeAgentsListFilter(filter)
	active := int(schema.ChatSessionActive)
	archivedOnly := filter == agentsListFilterArchived
	archived := archivedOnly
	opts := store.ListChatSessionsOptions{
		Limit:       20,
		Cursor:      cursor,
		UID:         uid,
		State:       &active,
		Archived:    &archived,
		PinnedFirst: !archivedOnly,
	}
	switch filter {
	case agentsListFilterRunning, agentsListFilterNeedsApproval:
		flags := chatagent.ListSessionIDsByActivity(filter)
		if len(flags) == 0 {
			return nil, "", nil
		}
		opts.Flags = flags
		opts.Cursor = ""
	}
	rows, nextCursor, err := store.Database.ListChatSessions(ctx.Context(), opts)
	if err != nil {
		return nil, "", err
	}
	leafBySession := make(map[string]string, len(rows))
	items := make([]model.AgentSession, 0, len(rows))
	for _, row := range rows {
		item := mapAgentSession(row)
		item.Activity = chatagent.SessionActivity(row.Flag)
		items = append(items, item)
		leafBySession[row.Flag] = row.LeafID
	}
	if len(leafBySession) > 0 {
		if durations, err := chatagent.SumSessionsRunDurationMs(ctx.Context(), leafBySession); err == nil {
			for i := range items {
				items[i].TotalDurationMs = durations[items[i].Flag]
			}
		}
	}
	if len(items) > 0 {
		sessionIDs := make([]string, len(items))
		for i := range items {
			sessionIDs[i] = items[i].Flag
		}
		if summaries, err := chatagent.SummarizeTodosBySessions(ctx.Context(), sessionIDs); err == nil {
			for i := range items {
				if summary, ok := summaries[items[i].Flag]; ok {
					items[i].TodoSummary = &summary
				}
			}
		}
	}
	return items, nextCursor, nil
}

func agentChatPin(ctx fiber.Ctx) error {
	return setAgentChatPinned(ctx, true)
}

func agentChatUnpin(ctx fiber.Ctx) error {
	return setAgentChatPinned(ctx, false)
}

func agentChatArchive(ctx fiber.Ctx) error {
	return setAgentChatArchived(ctx, true)
}

func agentChatUnarchive(ctx fiber.Ctx) error {
	return setAgentChatArchived(ctx, false)
}

func setAgentChatPinned(ctx fiber.Ctx, pinned bool) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return toastError(ctx, "Chat agent is not enabled")
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Session not found")
		}
		if errors.Is(err, types.ErrForbidden) {
			return toastError(ctx, "Forbidden")
		}
		return types.Errorf(types.ErrInternal, "pin session: %v", err)
	}
	if err := store.Database.UpdateChatSessionPinned(ctx.Context(), sessionID, pinned); err != nil {
		return toastError(ctx, "Failed to update pin")
	}
	filter := normalizeAgentsListFilter(ctx.Query("filter"))
	items, nextCursor, err := listUserAgentSessionModels(ctx, "", filter)
	if err != nil {
		return toastError(ctx, "Failed to refresh sessions")
	}
	ctx.Type("html")
	return partials.ChatAgentSessionList(items, nextCursor, agentsEndpointsWithFilter(filter)).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func setAgentChatArchived(ctx fiber.Ctx, archived bool) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := webRequireChatAgentEnabled(); err != nil {
		return toastError(ctx, "Chat agent is not enabled")
	}
	sessionID := strings.Clone(ctx.Params("id"))
	if err := ensureWebSessionOwner(ctx, sessionID); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Session not found")
		}
		if errors.Is(err, types.ErrForbidden) {
			return toastError(ctx, "Forbidden")
		}
		return types.Errorf(types.ErrInternal, "archive session: %v", err)
	}
	if err := chatagent.SetSessionArchived(ctx.Context(), sessionID, archived); err != nil {
		return toastError(ctx, "Failed to update archive")
	}
	filter := normalizeAgentsListFilter(ctx.Query("filter"))
	if archived && filter == agentsListFilterAll {
		filter = agentsListFilterAll
	}
	items, nextCursor, err := listUserAgentSessionModels(ctx, "", filter)
	if err != nil {
		return toastError(ctx, "Failed to refresh sessions")
	}
	ctx.Type("html")
	return partials.ChatAgentSessionList(items, nextCursor, agentsEndpointsWithFilter(filter)).
		Render(ctx.Context(), ctx.Response().BodyWriter())
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
		atts := make([]model.AgentChatAttachment, 0, len(m.Attachments))
		for _, a := range m.Attachments {
			atts = append(atts, model.AgentChatAttachment{
				FileID:   a.FileID,
				MIMEType: a.MIMEType,
				Kind:     a.Kind,
			})
		}
		return model.AgentChatMessage{
			Role:        "user",
			Kind:        "user",
			Text:        m.Text,
			HTML:        partials.FormatChatAgentMessageHTML("user", m.Text),
			Attachments: atts,
			CreatedAt:   m.CreatedAt,
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

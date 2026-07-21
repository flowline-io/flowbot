package server

import (
	"bufio"
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RegisterChatAgentRoutes wires Chat Agent REST endpoints for HTTP clients.
func RegisterChatAgentRoutes(a *fiber.App) {
	chatHTTP := newChatAgentHTTP()
	a.Get("/chatagent/info", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.info)))
	a.Get("/chatagent/sessions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listSessions)))
	a.Post("/chatagent/sessions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.createSession)))
	a.Delete("/chatagent/sessions/:id", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.closeSession)))
	a.Get("/chatagent/sessions/:id/messages", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listMessages)))
	a.Get("/chatagent/sessions/:id/plans", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listSessionPlans)))
	a.Get("/chatagent/sessions/:id/todos", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listSessionTodos)))
	a.Get("/chatagent/resources", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.getResource)))
	a.Get("/chatagent/sessions/:id/export", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.exportSession)))
	a.Get("/chatagent/sessions/:id/context", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.contextUsage)))
	a.Post("/chatagent/sessions/:id/compact", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.compactSession)))
	a.Post("/chatagent/sessions/:id/messages", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.sendMessage)))
	a.Post("/chatagent/sessions/:id/confirm", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.confirm)))
	a.Post("/chatagent/sessions/:id/cancel", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.cancelRun)))
	a.Get("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.getPermissions)))
	a.Put("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.putPermissions)))
	a.Delete("/chatagent/permissions", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.deletePermissions)))
	a.Get("/chatagent/sessions/:id/events", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.sessionEvents)))
	a.Get("/chatagent/sessions/:id/mode", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.getSessionMode)))
	a.Put("/chatagent/sessions/:id/mode", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.putSessionMode)))
	a.Delete("/chatagent/sessions/:id/permission-grants", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.clearPermissionGrants)))
	a.Get("/chatagent/scheduled-tasks", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listScheduledTasks)))
	a.Post("/chatagent/scheduled-tasks", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.createScheduledTask)))
	a.Get("/chatagent/scheduled-tasks/:id", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.getScheduledTask)))
	a.Patch("/chatagent/scheduled-tasks/:id", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.patchScheduledTask)))
	a.Delete("/chatagent/scheduled-tasks/:id", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.cancelScheduledTask)))
	a.Get("/chatagent/scheduled-tasks/:id/runs", route.Authorize(route.RequireScope(auth.ScopeChatAgentChat, chatHTTP.listScheduledTaskRuns)))

	if config.ChatAgentEnabled() {
		go func() {
			if err := chatagent.SeedDefaultSubagents(context.Background()); err != nil {
				flog.Warn("[chat-agent] seed default subagents: %v", err)
			}
		}()
	}
}

type chatAgentHTTP struct {
	service *chatagent.Service
}

func newChatAgentHTTP() *chatAgentHTTP {
	return &chatAgentHTTP{service: chatagent.NewService()}
}

func (*chatAgentHTTP) info(c fiber.Ctx) error {
	if err := requireChatAgentEnabled(); err != nil {
		return chatAgentError(c, err)
	}
	info, err := chatagent.BuildAgentInfo(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(info)
}

func (*chatAgentHTTP) ensureSessionOwner(c fiber.Ctx, sessionID string) error {
	rc := route.GetRequestContext(c)
	if rc == nil || rc.UID.IsZero() {
		return types.ErrUnauthorized
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	sess, err := store.Database.GetChatSession(c.Context(), sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.ErrNotFound
		}
		return err
	}
	if sess.UID != rc.UID.String() {
		return types.ErrForbidden
	}
	if sess.State == int(schema.ChatSessionClosed) {
		return types.ErrNotFound
	}
	return nil
}

func requireChatAgentEnabled() error {
	if !config.ChatAgentEnabled() {
		return chatagent.ErrChatAgentDisabled
	}
	return nil
}

func chatAgentError(c fiber.Ctx, err error) error {
	if errors.Is(err, chatagent.ErrChatAgentDisabled) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, chatagent.ErrRunInFlight) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if status, ok := domainErrorStatus(err); ok {
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	flog.Warn("[chat-agent] http error: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func writeChatAgentSSE(w *bufio.Writer, event chatagent.StreamEvent) bool {
	return (&chatagent.BufioSSEWriter{W: w}).WriteEvent(event)
}

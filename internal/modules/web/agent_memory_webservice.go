package web

import (
	"errors"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/agent/memory"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var agentMemoryWebserviceRules = []webservice.Rule{
	webservice.Get("/agent-memory/files", agentMemoryListFiles, route.WithNotAuth()),
	webservice.Get("/agent-memory/content", agentMemoryReadContent, route.WithNotAuth()),
	webservice.Put("/agent-memory/content", agentMemoryWriteContent, route.WithNotAuth()),
}

func openAgentMemoryStore() (*memory.FileStore, error) {
	return memory.OpenFromConfig()
}

type agentMemoryWriteRequest struct {
	Scope   string `json:"scope"`
	File    string `json:"file"`
	Content string `json:"content"`
}

func agentMemoryListFiles(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.Query("scope"))
	if scope == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope is required")))
	}
	store, err := openAgentMemoryStore()
	if err != nil {
		return types.Errorf(types.ErrInternal, "memory store: %v", err)
	}
	files, err := store.ListFiles(scope)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(files))
}

func agentMemoryReadContent(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	scope := strings.TrimSpace(ctx.Query("scope"))
	if scope == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope is required")))
	}
	file := strings.TrimSpace(ctx.Query("file"))
	store, err := openAgentMemoryStore()
	if err != nil {
		return types.Errorf(types.ErrInternal, "memory store: %v", err)
	}
	content, err := store.Read(scope, file)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{"content": content}))
}

func agentMemoryWriteContent(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	var req agentMemoryWriteRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("invalid JSON body")))
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(errors.New("scope is required")))
	}
	store, err := openAgentMemoryStore()
	if err != nil {
		return types.Errorf(types.ErrInternal, "memory store: %v", err)
	}
	if err := store.Write(scope, req.File, req.Content); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{"status": "saved"}))
}

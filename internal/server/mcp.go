package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/tool"
	"github.com/gofiber/fiber/v3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// hasToolRules checks if a bot has tool rules implemented.
func hasToolRules(botName string) bool {
	handler, ok := chatbot.List()[botName]
	if !ok || !handler.IsReady() {
		return false
	}

	for _, item := range handler.Rules() {
		if _, ok := item.([]tool.Rule); ok {
			return true
		}
	}
	return false
}

// getBotTools returns all tools for a specific bot.
func getBotTools(botName string, ctx types.Context) ([]llmTool.BaseTool, error) {
	handler, ok := chatbot.List()[botName]
	if !ok || !handler.IsReady() {
		return nil, fmt.Errorf("bot %s not found or not ready", botName)
	}

	var tools []llmTool.BaseTool
	for _, item := range handler.Rules() {
		if toolRules, ok := item.([]tool.Rule); ok {
			for _, rule := range toolRules {
				t, err := rule(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to create tool: %w", err)
				}
				tools = append(tools, t)
			}
		}
	}
	return tools, nil
}

// bearerTokenAuth middleware for BearerToken authentication.
// Validates Bearer token from Authorization header against configured MCP token.
func bearerTokenAuth(handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		authHeader := ctx.Get("Authorization")
		if authHeader == "" {
			return ctx.Status(fiber.StatusUnauthorized).
				JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized.New("missing authorization header")))
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return ctx.Status(fiber.StatusUnauthorized).
				JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized.New("invalid authorization header format")))
		}

		token := parts[1]
		if token == "" {
			return ctx.Status(fiber.StatusUnauthorized).
				JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized.New("missing bearer token")))
		}

		// Validate token against configured MCP token
		validToken := config.App.Flowbot.MCPToken
		if validToken != "" && token != validToken {
			return ctx.Status(fiber.StatusUnauthorized).
				JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized.New("invalid bearer token")))
		}

		// Store token in context for later use
		ctx.Locals("mcp_token", token)

		return handler(ctx)
	}
}

// mcpServerManager manages MCP servers for each bot with thread-safe access.
type mcpServerManager struct {
	mu      sync.RWMutex
	servers map[string]*mcp.Server
}

var serverManager = &mcpServerManager{
	servers: make(map[string]*mcp.Server),
}

// get retrieves an MCP server for a bot (thread-safe).
func (m *mcpServerManager) get(botName string) (*mcp.Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[botName]
	return server, ok
}

// set stores an MCP server for a bot (thread-safe).
func (m *mcpServerManager) set(botName string, server *mcp.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[botName] = server
}

// getOrCreateMCPServer creates or returns an MCP server for a bot.
// Uses double-checked locking pattern to prevent race conditions.
func getOrCreateMCPServer(botName string) (*mcp.Server, error) {
	// First check (fast path, read-only)
	if server, ok := serverManager.get(botName); ok {
		return server, nil
	}

	// Acquire write lock for creation
	serverManager.mu.Lock()
	defer serverManager.mu.Unlock()

	// Second check (another goroutine might have created it)
	if server, ok := serverManager.servers[botName]; ok {
		return server, nil
	}

	// Create new MCP server using SDK
	server := mcp.NewServer(&mcp.Implementation{
		Name:    botName,
		Version: "1.0.0",
	}, nil)

	// Register tools for this bot
	typesCtx := types.Context{}
	tools, err := getBotTools(botName, typesCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot tools: %w", err)
	}

	// Register tools, continue even if some fail
	toolCount := 0
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			flog.Warn("failed to get tool info for bot %s: %v", botName, err)
			continue
		}

		inputSchema := convertParamsOneOfToJSONSchema(info.ParamsOneOf)
		server.AddTool(&mcp.Tool{
			Name:        info.Name,
			Description: info.Desc,
			InputSchema: inputSchema,
		}, createToolHandler(botName, info.Name))
		toolCount++
	}

	if toolCount == 0 {
		flog.Warn("no tools registered for bot %s", botName)
	}

	// Store server
	serverManager.servers[botName] = server
	return server, nil
}

// convertParamsOneOfToJSONSchema converts ParamsOneOf to JSON Schema format.
func convertParamsOneOfToJSONSchema(paramsOneOf *schema.ParamsOneOf) interface{} {
	if paramsOneOf == nil {
		return map[string]interface{}{
			"type": "object",
		}
	}

	// Use ParamsOneOf's ToJSONSchema method
	schemaObj, err := paramsOneOf.ToJSONSchema()
	if err != nil {
		flog.Warn("failed to convert ParamsOneOf to JSON Schema: %v", err)
		return map[string]interface{}{
			"type": "object",
		}
	}

	if schemaObj == nil {
		return map[string]interface{}{
			"type": "object",
		}
	}

	// Convert jsonschema.Schema to map for JSON marshaling
	data, err := json.Marshal(schemaObj)
	if err != nil {
		flog.Warn("failed to marshal JSON Schema: %v", err)
		return map[string]interface{}{
			"type": "object",
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		flog.Warn("failed to unmarshal JSON Schema: %v", err)
		return map[string]interface{}{
			"type": "object",
		}
	}

	// If schema is empty or invalid, return default
	if len(result) == 0 {
		return map[string]interface{}{
			"type": "object",
		}
	}

	return result
}

// executeTool executes a tool with the given bot name, tool name, and arguments.
// This is a shared function used by both SDK handler and fallback handler.
func executeTool(botName, toolName string, argumentsJSON string) (string, error) {
	// Create context for tool execution
	typesCtx := types.Context{
		ToolRuleId: toolName,
	}
	typesCtx.SetTimeout(30 * time.Second)

	// Get bot handler
	handler, ok := chatbot.List()[botName]
	if !ok || !handler.IsReady() {
		return "", fmt.Errorf("bot %s not found or not ready", botName)
	}

	// Execute tool using the context
	result, err := handler.Tool(typesCtx, argumentsJSON)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// createToolHandler creates a handler function for a specific tool.
func createToolHandler(botName, toolName string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert arguments to JSON string
		argumentsJSON, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("failed to marshal arguments: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Execute tool using shared function
		result, err := executeTool(botName, toolName, string(argumentsJSON))
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("tool execution failed: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Return result in MCP format
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: result,
				},
			},
			IsError: false,
		}, nil
	}
}

// mcpHandler handles MCP protocol requests using the MCP SDK.
// Currently uses fallback implementation as MCP SDK requires Transport interface
// which is designed for bidirectional connections, while HTTP is request-response.
func (c *Controller) mcpHandler(ctx fiber.Ctx) error {
	botName := ctx.Params("bot_name")
	if botName == "" {
		return ctx.Status(fiber.StatusBadRequest).
			JSON(protocol.NewFailedResponse(protocol.ErrBadParam.New("bot_name parameter is required")))
	}

	// Check if bot has tool rules
	if !hasToolRules(botName) {
		return ctx.Status(fiber.StatusNotFound).
			JSON(protocol.NewFailedResponse(protocol.ErrNotFound.New(fmt.Sprintf("bot %s does not implement tool rules", botName))))
	}

	// Get or create MCP server for this bot (for future use with proper Transport)
	_, err := getOrCreateMCPServer(botName)
	if err != nil {
		flog.Warn("failed to create MCP server for bot %s: %v", botName, err)
		// Continue with fallback even if server creation fails
	}

	// Use fallback implementation for HTTP request-response model
	// TODO: Implement proper HTTP Transport adapter for MCP SDK if needed
	return c.handleMCPRequestFallback(ctx, botName)
}

// handleMCPRequestFallback handles MCP requests using fallback implementation.
// This is used because MCP SDK's Transport interface is designed for bidirectional
// connections, while HTTP follows a request-response model.
func (c *Controller) handleMCPRequestFallback(ctx fiber.Ctx, botName string) error {
	method := ctx.Method()

	switch method {
	case http.MethodPost:
		return c.handleMCPToolCall(ctx, botName)
	case http.MethodGet:
		return c.handleMCPListTools(ctx, botName)
	default:
		return ctx.Status(fiber.StatusMethodNotAllowed).
			JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.New("method not allowed")))
	}
}

// handleMCPToolCall handles MCP tool call requests.
func (c *Controller) handleMCPToolCall(ctx fiber.Ctx, botName string) error {
	var mcpRequest map[string]interface{}
	if err := json.Unmarshal(ctx.Body(), &mcpRequest); err != nil {
		return ctx.Status(fiber.StatusBadRequest).
			JSON(protocol.NewFailedResponse(protocol.ErrBadParam.Wrap(err)))
	}

	// Extract tool name and arguments from MCP request
	// MCP protocol uses "name" for tool name
	toolName, _ := mcpRequest["name"].(string)
	if toolName == "" {
		// Try alternative field name for compatibility
		toolName, _ = mcpRequest["tool"].(string)
	}

	arguments, _ := mcpRequest["arguments"].(map[string]interface{})

	if toolName == "" {
		return ctx.Status(fiber.StatusBadRequest).
			JSON(protocol.NewFailedResponse(protocol.ErrBadParam.New("tool name is required")))
	}

	// Convert arguments to JSON string
	argumentsJSON, err := json.Marshal(arguments)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).
			JSON(protocol.NewFailedResponse(protocol.ErrBadParam.Wrap(err)))
	}

	// Execute tool using shared function
	result, err := executeTool(botName, toolName, string(argumentsJSON))
	if err != nil {
		flog.Error(fmt.Errorf("MCP tool execution failed: %w", err))
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(protocol.NewFailedResponse(protocol.ErrInternalServerError.Wrap(err)))
	}

	// Return MCP response format
	return ctx.JSON(fiber.Map{
		"content": []fiber.Map{
			{
				"type": "text",
				"text": result,
			},
		},
		"isError": false,
	})
}

// handleMCPListTools returns available tools for the bot in MCP format.
func (c *Controller) handleMCPListTools(ctx fiber.Ctx, botName string) error {
	typesCtx := types.Context{}

	// Get all tools for the bot
	tools, err := getBotTools(botName, typesCtx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(protocol.NewFailedResponse(protocol.ErrInternalServerError.Wrap(err)))
	}

	// Convert tools to MCP format
	mcpTools := make([]fiber.Map, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			flog.Warn("failed to get tool info: %v", err)
			continue
		}

		// Convert tool schema to MCP format
		inputSchema := convertParamsOneOfToJSONSchema(info.ParamsOneOf)
		mcpTool := fiber.Map{
			"name":        info.Name,
			"description": info.Desc,
			"inputSchema": inputSchema,
		}

		mcpTools = append(mcpTools, mcpTool)
	}

	// Return MCP format response
	return ctx.JSON(fiber.Map{
		"tools": mcpTools,
	})
}

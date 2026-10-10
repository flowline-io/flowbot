package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

type remoteTool struct {
	server string
	remote string
	desc   string
	schema map[string]any
	call   func(ctx context.Context, name string, args map[string]any) (*mcpsdk.CallToolResult, error)
}

// Name returns mcp_<server>_<remote>.
func (t remoteTool) Name() string {
	return pkgmcp.AgentToolName(t.server, t.remote)
}

// Description explains the remote tool.
func (t remoteTool) Description() string {
	if t.desc != "" {
		return t.desc
	}
	return "MCP tool " + t.remote + " from server " + t.server
}

// Parameters returns the remote JSON schema.
func (t remoteTool) Parameters() map[string]any {
	if t.schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return t.schema
}

// Execute calls the remote MCP tool.
func (t remoteTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if t.call == nil {
		return tool.ErrorResult(id, t.Name(), "unavailable", "mcp session not connected", "check chat_agent.mcp_servers"), nil
	}
	result, err := t.call(ctx, t.remote, args)
	if err != nil {
		return tool.ErrorResult(id, t.Name(), "mcp_error", err.Error(), "retry or inspect the MCP server logs"), nil
	}
	text := formatCallResult(result)
	if result != nil && result.IsError {
		return tool.ErrorResult(id, t.Name(), "mcp_tool_error", text, "inspect the MCP tool error"), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

func formatCallResult(result *mcpsdk.CallToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, c := range result.Content {
		if texter, ok := c.(*mcpsdk.TextContent); ok {
			parts = append(parts, texter.Text)
			continue
		}
		raw, err := sonic.Marshal(c)
		if err != nil {
			parts = append(parts, fmt.Sprint(c))
			continue
		}
		parts = append(parts, string(raw))
	}
	if result.StructuredContent != nil {
		raw, err := sonic.Marshal(result.StructuredContent)
		if err == nil && string(raw) != "null" {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, "\n")
}

func schemaMap(raw any) map[string]any {
	if raw == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	if m, ok := raw.(map[string]any); ok {
		return m
	}
	b, err := sonic.Marshal(raw)
	if err != nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	var m map[string]any
	if err := sonic.Unmarshal(b, &m); err != nil || m == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return m
}

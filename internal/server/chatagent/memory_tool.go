package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/memory"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
)

// UpdateMemoryTool reads and writes persistent memory markdown files outside the workspace.
type UpdateMemoryTool struct {
	Store *memory.FileStore
}

// NewUpdateMemoryTool builds a tool backed by the configured memory directory.
func NewUpdateMemoryTool() (UpdateMemoryTool, error) {
	dir, err := config.MemoryDirectory()
	if err != nil {
		return UpdateMemoryTool{}, err
	}
	store, err := memory.NewFileStore(
		dir,
		config.ChatAgentDefaultMemoryFile(),
		config.ChatAgentMemoryMaxFileBytes(),
	)
	if err != nil {
		return UpdateMemoryTool{}, err
	}
	return UpdateMemoryTool{Store: store}, nil
}

// Name returns the tool identifier.
func (UpdateMemoryTool) Name() string { return updateMemoryToolName }

// Description explains the tool to the model.
func (UpdateMemoryTool) Description() string {
	return "Read, write, or list persistent memory markdown files stored outside the workspace across automation runs"
}

// Parameters returns the JSON schema for tool arguments.
func (UpdateMemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "One of read, write, or list",
				"enum":        []string{"read", "write", "list"},
			},
			"file": map[string]any{
				"type":        "string",
				"description": "Memory markdown filename; defaults to MEMORIES.md",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Markdown content to write when operation is write",
			},
		},
		"required": []string{"operation"},
	}
}

// Execute runs one memory operation.
func (t UpdateMemoryTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if t.Store == nil {
		return memoryToolError(id, "memory store is not configured"), nil
	}
	operation := strings.ToLower(strings.TrimSpace(MemoryOperation(args)))
	switch operation {
	case "read":
		return t.executeRead(ctx, id, args)
	case "write":
		return t.executeWrite(ctx, id, args)
	case "list":
		return t.executeList(ctx, id)
	default:
		return memoryToolError(id, "operation is required and must be read, write, or list"), nil
	}
}

func (t UpdateMemoryTool) executeRead(ctx context.Context, id string, args map[string]any) (msg.ToolResultMessage, error) {
	scope := MemoryScopeFromContext(ctx)
	content, err := t.Store.Read(scope, memoryFileArg(args))
	if err != nil {
		return memoryToolError(id, err.Error()), nil
	}
	if strings.TrimSpace(content) == "" {
		content = "(empty)"
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       updateMemoryToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: content}},
	}, nil
}

func (t UpdateMemoryTool) executeWrite(ctx context.Context, id string, args map[string]any) (msg.ToolResultMessage, error) {
	content := memoryContentArg(args)
	if strings.TrimSpace(content) == "" {
		return memoryToolError(id, "content is required for write"), nil
	}
	scope := MemoryScopeFromContext(ctx)
	if err := t.Store.Write(scope, memoryFileArg(args), content); err != nil {
		return memoryToolError(id, err.Error()), nil
	}
	file := memoryFileArg(args)
	if file == "" {
		file = t.Store.DefaultFile
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       updateMemoryToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: fmt.Sprintf("saved %s", file)}},
	}, nil
}

func (t UpdateMemoryTool) executeList(ctx context.Context, id string) (msg.ToolResultMessage, error) {
	scope := MemoryScopeFromContext(ctx)
	files, err := t.Store.ListFiles(scope)
	if err != nil {
		return memoryToolError(id, err.Error()), nil
	}
	text := "(none)"
	if len(files) > 0 {
		text = strings.Join(files, "\n")
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       updateMemoryToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

func memoryToolError(id, message string) msg.ToolResultMessage {
	return tool.ErrorResult(id, updateMemoryToolName, "memory_error", message, "")
}

func memoryFileArg(args map[string]any) string {
	if args == nil {
		return ""
	}
	v, ok := args["file"]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func memoryContentArg(args map[string]any) string {
	if args == nil {
		return ""
	}
	v, ok := args["content"]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

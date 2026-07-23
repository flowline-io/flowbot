package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	memorySetToolName              = "memory_set"
	memoryGetToolName              = "memory_get"
	memoryListToolName             = "memory_list"
	memoryDeleteToolName           = "memory_delete"
	searchSessionSummariesToolName = "search_session_summaries"
	defaultMemoryFactKeyMaxLen     = 128
)

// MemoryToolNames returns fact and session-summary memory tools.
func MemoryToolNames() []string {
	return []string{
		memorySetToolName,
		memoryGetToolName,
		memoryListToolName,
		memoryDeleteToolName,
		searchSessionSummariesToolName,
	}
}

// RegisterMemoryTools registers DB-backed memory tools on registry.
func RegisterMemoryTools(registry *tool.Registry) error {
	for _, t := range []tool.Tool{
		MemorySetTool{},
		MemoryGetTool{},
		MemoryListTool{},
		MemoryDeleteTool{},
		SearchSessionSummariesTool{},
	} {
		if err := registry.Register(t); err != nil {
			return err
		}
	}
	return nil
}

func resolveToolMemoryScope(ctx context.Context) string {
	scope := strings.TrimSpace(MemoryScopeFromContext(ctx))
	if scope == "" {
		return "default"
	}
	return scope
}

func memoryStringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func memoryBoolArg(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && b
	default:
		return false
	}
}

func memoryIntArg(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	switch v := args[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func validateMemoryKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is required")
	}
	if len(key) > defaultMemoryFactKeyMaxLen {
		return fmt.Errorf("key exceeds %d characters", defaultMemoryFactKeyMaxLen)
	}
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return errors.New("key may only contain letters, digits, '.', '_' and '-'")
	}
	return nil
}

func memoryToolError(name, id, message string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		IsError:    true,
		Parts:      []msg.ContentPart{msg.TextPart{Text: tool.FormatToolError("memory", message, "")}},
	}
}

func memoryToolText(name, id, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}
}

// MemorySetTool upserts one keyed fact in the active memory scope.
type MemorySetTool struct{}

// Name returns the tool identifier.
func (MemorySetTool) Name() string { return memorySetToolName }

// Description explains the tool to the model.
func (MemorySetTool) Description() string {
	return "Save or update a keyed memory fact for the current memory scope (shared across interactive chats when scope is default)"
}

// Parameters returns the JSON schema for tool arguments.
func (MemorySetTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key": map[string]any{
				"type":        "string",
				"description": "Fact key (e.g. user.name or pref.language)",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Fact value to store",
			},
			"pinned": map[string]any{
				"type":        "boolean",
				"description": "When true, prefer this fact for automatic system-prompt injection",
			},
		},
		"required": []string{"key", "value"},
	}
}

// Execute upserts a memory fact.
func (MemorySetTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return memoryToolError(memorySetToolName, id, "memory store is not configured"), nil
	}
	key := memoryStringArg(args, "key")
	if err := validateMemoryKey(key); err != nil {
		return memoryToolError(memorySetToolName, id, err.Error()), nil
	}
	value := memoryStringArg(args, "value")
	if value == "" {
		return memoryToolError(memorySetToolName, id, "value is required"), nil
	}
	scope := resolveToolMemoryScope(ctx)
	row, err := store.Database.UpsertAgentMemoryFact(ctx, store.AgentMemoryFactUpsert{
		Scope:  scope,
		Key:    key,
		Value:  value,
		Pinned: memoryBoolArg(args, "pinned"),
	})
	if err != nil {
		return memoryToolError(memorySetToolName, id, err.Error()), nil
	}
	return memoryToolText(memorySetToolName, id, fmt.Sprintf("saved %s/%s (pinned=%t)", scope, row.Key, row.Pinned)), nil
}

// MemoryGetTool reads one keyed fact.
type MemoryGetTool struct{}

// Name returns the tool identifier.
func (MemoryGetTool) Name() string { return memoryGetToolName }

// Description explains the tool to the model.
func (MemoryGetTool) Description() string {
	return "Read one memory fact by key from the current memory scope"
}

// Parameters returns the JSON schema for tool arguments.
func (MemoryGetTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key": map[string]any{
				"type":        "string",
				"description": "Fact key to read",
			},
		},
		"required": []string{"key"},
	}
}

// Execute reads a memory fact.
func (MemoryGetTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return memoryToolError(memoryGetToolName, id, "memory store is not configured"), nil
	}
	key := memoryStringArg(args, "key")
	if err := validateMemoryKey(key); err != nil {
		return memoryToolError(memoryGetToolName, id, err.Error()), nil
	}
	row, err := store.Database.GetAgentMemoryFact(ctx, resolveToolMemoryScope(ctx), key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return memoryToolError(memoryGetToolName, id, "fact not found"), nil
		}
		return memoryToolError(memoryGetToolName, id, err.Error()), nil
	}
	payload, err := sonic.MarshalString(map[string]any{
		"key":    row.Key,
		"value":  row.Value,
		"pinned": row.Pinned,
	})
	if err != nil {
		return memoryToolError(memoryGetToolName, id, err.Error()), nil
	}
	return memoryToolText(memoryGetToolName, id, payload), nil
}

// MemoryListTool lists keys in the active scope.
type MemoryListTool struct{}

// Name returns the tool identifier.
func (MemoryListTool) Name() string { return memoryListToolName }

// Description explains the tool to the model.
func (MemoryListTool) Description() string {
	return "List memory fact keys (and pin flags) in the current memory scope"
}

// Parameters returns the JSON schema for tool arguments.
func (MemoryListTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute lists memory facts.
func (MemoryListTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return memoryToolError(memoryListToolName, id, "memory store is not configured"), nil
	}
	rows, err := store.Database.ListAgentMemoryFacts(ctx, resolveToolMemoryScope(ctx))
	if err != nil {
		return memoryToolError(memoryListToolName, id, err.Error()), nil
	}
	type item struct {
		Key    string `json:"key"`
		Pinned bool   `json:"pinned"`
	}
	items := make([]item, 0, len(rows))
	for _, row := range rows {
		items = append(items, item{Key: row.Key, Pinned: row.Pinned})
	}
	if len(items) == 0 {
		return memoryToolText(memoryListToolName, id, "[]"), nil
	}
	payload, err := sonic.MarshalString(items)
	if err != nil {
		return memoryToolError(memoryListToolName, id, err.Error()), nil
	}
	return memoryToolText(memoryListToolName, id, payload), nil
}

// MemoryDeleteTool deletes one keyed fact.
type MemoryDeleteTool struct{}

// Name returns the tool identifier.
func (MemoryDeleteTool) Name() string { return memoryDeleteToolName }

// Description explains the tool to the model.
func (MemoryDeleteTool) Description() string {
	return "Delete one memory fact by key from the current memory scope"
}

// Parameters returns the JSON schema for tool arguments.
func (MemoryDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key": map[string]any{
				"type":        "string",
				"description": "Fact key to delete",
			},
		},
		"required": []string{"key"},
	}
}

// Execute deletes a memory fact.
func (MemoryDeleteTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return memoryToolError(memoryDeleteToolName, id, "memory store is not configured"), nil
	}
	key := memoryStringArg(args, "key")
	if err := validateMemoryKey(key); err != nil {
		return memoryToolError(memoryDeleteToolName, id, err.Error()), nil
	}
	scope := resolveToolMemoryScope(ctx)
	if err := store.Database.DeleteAgentMemoryFact(ctx, scope, key); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return memoryToolError(memoryDeleteToolName, id, "fact not found"), nil
		}
		return memoryToolError(memoryDeleteToolName, id, err.Error()), nil
	}
	return memoryToolText(memoryDeleteToolName, id, fmt.Sprintf("deleted %s/%s", scope, key)), nil
}

// SearchSessionSummariesTool searches archived session summaries.
type SearchSessionSummariesTool struct{}

// Name returns the tool identifier.
func (SearchSessionSummariesTool) Name() string { return searchSessionSummariesToolName }

// Description explains the tool to the model.
func (SearchSessionSummariesTool) Description() string {
	return "Search archived chat session summaries by keyword (title/summary substring match)"
}

// Parameters returns the JSON schema for tool arguments.
func (SearchSessionSummariesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query matched against summary title and body",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max results (default 10, max 50)",
			},
		},
		"required": []string{"query"},
	}
}

// Execute searches session summaries in the active memory scope.
func (SearchSessionSummariesTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return memoryToolError(searchSessionSummariesToolName, id, "memory store is not configured"), nil
	}
	query := memoryStringArg(args, "query")
	if query == "" {
		return memoryToolError(searchSessionSummariesToolName, id, "query is required"), nil
	}
	rows, err := store.Database.SearchAgentSessionSummaries(ctx, store.AgentSessionSummarySearchParams{
		Query: query,
		Scope: resolveToolMemoryScope(ctx),
		Limit: memoryIntArg(args, "limit"),
	})
	if err != nil {
		return memoryToolError(searchSessionSummariesToolName, id, err.Error()), nil
	}
	type hit struct {
		SessionFlag string `json:"session_flag"`
		Title       string `json:"title"`
		Summary     string `json:"summary"`
	}
	hits := make([]hit, 0, len(rows))
	for _, row := range rows {
		hits = append(hits, hit{
			SessionFlag: row.SessionFlag,
			Title:       row.Title,
			Summary:     row.Summary,
		})
	}
	if len(hits) == 0 {
		return memoryToolText(searchSessionSummariesToolName, id, "[]"), nil
	}
	payload, err := sonic.MarshalString(hits)
	if err != nil {
		return memoryToolError(searchSessionSummariesToolName, id, err.Error()), nil
	}
	return memoryToolText(searchSessionSummariesToolName, id, payload), nil
}

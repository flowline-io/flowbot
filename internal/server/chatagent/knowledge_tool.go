package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	searchKnowledgeToolName = "search_knowledge"
	getKnowledgeToolName    = "get_knowledge"
)

// SearchKnowledgeTool searches the global knowledge base and returns metadata hits.
type SearchKnowledgeTool struct{}

// Name returns the tool identifier.
func (SearchKnowledgeTool) Name() string { return searchKnowledgeToolName }

// Description explains the tool to the model.
func (SearchKnowledgeTool) Description() string {
	return "Search the knowledge base for markdown documents; returns path, title, tags, and summary (not full content)"
}

// Parameters returns the JSON schema for tool arguments.
func (SearchKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Full-text query matched against path, title, tags, summary, and content",
			},
			"path_prefix": map[string]any{
				"type":        "string",
				"description": "Optional path prefix filter (e.g. /docs/develop/)",
			},
			"tag": map[string]any{
				"type":        "string",
				"description": "Optional exact tag filter",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max results (default 10, max 50)",
			},
		},
	}
}

// Execute runs a knowledge search.
func (SearchKnowledgeTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return knowledgeToolError(searchKnowledgeToolName, id, "knowledge store is not configured"), nil
	}
	query := knowledgeStringArg(args, "query")
	prefix := knowledgeStringArg(args, "path_prefix")
	if query == "" && prefix == "" {
		return knowledgeToolError(searchKnowledgeToolName, id, "query or path_prefix is required"), nil
	}
	limit := knowledgeIntArg(args, "limit")
	rows, err := store.Database.SearchAgentKnowledge(ctx, store.AgentKnowledgeSearchParams{
		Query:      query,
		PathPrefix: prefix,
		Tag:        knowledgeStringArg(args, "tag"),
		Limit:      limit,
	})
	if err != nil {
		return knowledgeToolError(searchKnowledgeToolName, id, err.Error()), nil
	}
	type hit struct {
		Path    string   `json:"path"`
		Title   string   `json:"title"`
		Tags    []string `json:"tags"`
		Summary string   `json:"summary"`
	}
	hits := make([]hit, 0, len(rows))
	for _, row := range rows {
		tags := row.Tags
		if tags == nil {
			tags = []string{}
		}
		hits = append(hits, hit{
			Path:    row.Path,
			Title:   row.Title,
			Tags:    tags,
			Summary: row.Summary,
		})
	}
	if len(hits) == 0 {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       searchKnowledgeToolName,
			Parts:      []msg.ContentPart{msg.TextPart{Text: "(no matches)"}},
		}, nil
	}
	payload, err := sonic.Marshal(hits)
	if err != nil {
		return knowledgeToolError(searchKnowledgeToolName, id, err.Error()), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       searchKnowledgeToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: string(payload)}},
	}, nil
}

// GetKnowledgeTool loads one knowledge document by path.
type GetKnowledgeTool struct {
	MaxOutput int
}

// Name returns the tool identifier.
func (GetKnowledgeTool) Name() string { return getKnowledgeToolName }

// Description explains the tool to the model.
func (GetKnowledgeTool) Description() string {
	return "Read a knowledge base markdown document by its path (use search_knowledge first to find paths)"
}

// Parameters returns the JSON schema for tool arguments.
func (GetKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Knowledge document path (e.g. /docs/develop/api-specs.md)",
			},
		},
		"required": []string{"path"},
	}
}

// Execute loads one document by path.
func (t GetKnowledgeTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if store.Database == nil {
		return knowledgeToolError(getKnowledgeToolName, id, "knowledge store is not configured"), nil
	}
	path := knowledgeStringArg(args, "path")
	if err := ValidateKnowledgePath(path); err != nil {
		return knowledgeToolError(getKnowledgeToolName, id, err.Error()), nil
	}
	row, err := store.Database.GetAgentKnowledgeByPath(ctx, path)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return knowledgeToolError(getKnowledgeToolName, id, "document not found"), nil
		}
		return knowledgeToolError(getKnowledgeToolName, id, err.Error()), nil
	}
	ws := coding.Workspace{MaxOutput: t.maxOutput()}
	body := fmt.Sprintf("# %s\n\npath: %s\n\n%s", row.Title, row.Path, row.Content)
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       getKnowledgeToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: ws.TruncateOutput(body)}},
	}, nil
}

func (t GetKnowledgeTool) maxOutput() int {
	if t.MaxOutput > 0 {
		return t.MaxOutput
	}
	maxOutput := config.App.ChatAgent.MaxToolOutput
	if maxOutput <= 0 {
		return 8192
	}
	return maxOutput
}

func knowledgeToolError(name, id, message string) msg.ToolResultMessage {
	return tool.ErrorResult(id, name, "knowledge_error", message, "")
}

func knowledgeStringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func knowledgeIntArg(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0
		}
		return parsed
	default:
		parsed, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
		if err != nil {
			return 0
		}
		return parsed
	}
}

// KnowledgeToolNames returns the knowledge tool identifiers.
func KnowledgeToolNames() []string {
	return []string{searchKnowledgeToolName, getKnowledgeToolName}
}

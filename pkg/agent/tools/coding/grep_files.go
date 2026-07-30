package coding

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// GrepFilesTool searches file contents with a regular expression.
type GrepFilesTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (GrepFilesTool) Name() string { return "grep_files" }

// Description explains the tool to the model.
func (GrepFilesTool) Description() string {
	return "Searches workspace file contents with a Go regular expression; returns path:line:text matches"
}

// Parameters returns the JSON schema for tool arguments.
func (GrepFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Go regular expression to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional search root relative to the workspace (default .)",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Optional path glob filter (supports **)",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "When true, match without case sensitivity",
			},
			"max_matches": map[string]any{
				"type":        "integer",
				"description": "Maximum matches to return",
			},
		},
		"required": []string{"pattern"},
	}
}

// Execute runs the content search.
func (t GrepFilesTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	pattern := fmt.Sprint(args["pattern"])
	if strings.TrimSpace(pattern) == "" || pattern == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "pattern is required", "provide a regular expression"), nil
	}
	if boolArg(args, "case_insensitive") {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return tool.ErrorResult(id, t.Name(), "invalid_args", fmt.Sprintf("invalid regexp: %v", err), "fix the pattern syntax"), nil
	}

	pathArg := normalizeWorkspacePath(fmt.Sprint(args["path"]))
	if pathArg == "" || pathArg == "<nil>" {
		pathArg = "."
	}
	globFilter := strings.TrimSpace(fmt.Sprint(args["glob"]))
	if globFilter == "<nil>" {
		globFilter = ""
	}
	maxMatches := ClampMaxMatches(intArg(args, "max_matches"), DefaultGrepMaxMatches, HardGrepMaxMatches)

	resolvedResult := t.Workspace.ResolvePath(pathArg)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}
	searchRoot := resolvedResult.Value()
	wsRoot := t.Workspace.absRoot()
	if !wsRoot.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(wsRoot.ErrorValue())), nil
	}

	hits, truncReason, err := walkGrep(ctx, t.executionEnv(), wsRoot.Value(), searchRoot, re, globFilter, maxMatches)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}
	if len(hits) == 0 {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       t.Name(),
			Parts:      []msg.ContentPart{msg.TextPart{Text: "No matches found."}},
		}, nil
	}
	text := strings.Join(hits, "\n")
	if truncReason != "" {
		text += "\n..." + truncReason
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(text)}},
	}, nil
}

func (t GrepFilesTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

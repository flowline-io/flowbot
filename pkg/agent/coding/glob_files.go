package coding

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// GlobFilesTool finds files by path pattern inside the workspace.
type GlobFilesTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (GlobFilesTool) Name() string { return "glob_files" }

// Description explains the tool to the model.
func (GlobFilesTool) Description() string {
	return "Finds files by glob pattern (supports **); returns relative paths only"
}

// Parameters returns the JSON schema for tool arguments.
func (GlobFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern such as **/*.go",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional search root relative to the workspace (default .)",
			},
			"max_matches": map[string]any{
				"type":        "integer",
				"description": "Maximum paths to return",
			},
		},
		"required": []string{"pattern"},
	}
}

// Execute runs the glob search.
func (t GlobFilesTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	pattern := strings.TrimSpace(fmt.Sprint(args["pattern"]))
	if pattern == "" || pattern == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "pattern is required", "provide a glob pattern such as **/*.go"), nil
	}
	pathArg := normalizeWorkspacePath(fmt.Sprint(args["path"]))
	if pathArg == "" || pathArg == "<nil>" {
		pathArg = "."
	}
	maxMatches := ClampMaxMatches(intArg(args, "max_matches"), DefaultGlobMaxMatches, HardGlobMaxMatches)

	resolvedResult := t.Workspace.ResolvePath(pathArg)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}
	searchRoot := resolvedResult.Value()
	absWorkspace := t.Workspace.absRoot()
	if !absWorkspace.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(absWorkspace.ErrorValue())), nil
	}

	matches, truncated, err := walkGlob(ctx, t.executionEnv(), absWorkspace.Value(), searchRoot, pattern, maxMatches)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}
	if len(matches) == 0 {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       t.Name(),
			Parts:      []msg.ContentPart{msg.TextPart{Text: "No files matched."}},
		}, nil
	}
	text := strings.Join(matches, "\n")
	if truncated {
		text += fmt.Sprintf("\n...(truncated to %d matches)", maxMatches)
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(text)}},
	}, nil
}

func (t GlobFilesTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func walkGlob(ctx context.Context, execEnv env.ExecutionEnv, workspaceRoot, searchRoot, pattern string, maxMatches int) ([]string, bool, error) {
	var matches []string
	truncated := false
	var walk func(string) error
	walk = func(dir string) error {
		if len(matches) >= maxMatches {
			truncated = true
			return nil
		}
		entriesResult := execEnv.ReadDir(ctx, dir)
		if !entriesResult.IsOk() {
			return fmt.Errorf("%s", env.FormatFileError(entriesResult.ErrorValue()))
		}
		for _, entry := range entriesResult.Value() {
			if len(matches) >= maxMatches {
				truncated = true
				return nil
			}
			abs := filepath.Join(dir, entry.Name)
			if entry.IsDir {
				if ShouldSkipDir(entry.Name) {
					continue
				}
				if err := walk(abs); err != nil {
					return err
				}
				continue
			}
			rel, err := filepath.Rel(workspaceRoot, abs)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			ok, err := MatchPath(pattern, rel)
			if err != nil {
				return err
			}
			if !ok {
				// Also try matching relative to search root when pattern has no ** prefix.
				relSearch, err := filepath.Rel(searchRoot, abs)
				if err != nil {
					return err
				}
				relSearch = filepath.ToSlash(relSearch)
				ok, err = MatchPath(pattern, relSearch)
				if err != nil {
					return err
				}
			}
			if ok {
				matches = append(matches, rel)
			}
		}
		return nil
	}
	if err := walk(searchRoot); err != nil {
		return nil, false, err
	}
	return matches, truncated, nil
}

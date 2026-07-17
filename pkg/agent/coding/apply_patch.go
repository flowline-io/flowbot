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

// ApplyPatchTool applies an OpenAI/Codex-style patch inside the workspace.
type ApplyPatchTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (ApplyPatchTool) Name() string { return "apply_patch" }

// Description explains the tool to the model.
func (ApplyPatchTool) Description() string {
	return "Applies a *** Begin Patch *** End Patch diff to add, update, or delete workspace files; validates all hunks before writing and rolls back on failure"
}

// Parameters returns the JSON schema for tool arguments.
func (ApplyPatchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"patch": map[string]any{
				"type":        "string",
				"description": "Codex-style patch text beginning with *** Begin Patch",
			},
		},
		"required": []string{"patch"},
	}
}

type patchOpKind string

const (
	patchAdd    patchOpKind = "add"
	patchUpdate patchOpKind = "update"
	patchDelete patchOpKind = "delete"
)

type patchOp struct {
	Kind    patchOpKind
	Path    string
	Content string
	Hunks   [][]string
}

type plannedPatch struct {
	Kind    patchOpKind
	Path    string
	Abs     string
	Content []byte
}

// Execute applies the patch with all-or-nothing semantics.
func (t ApplyPatchTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	ops, errResult := t.parseArgs(id, args)
	if errResult != nil {
		return *errResult, nil
	}
	planned, errResult := t.planOps(ctx, id, ops)
	if errResult != nil {
		return *errResult, nil
	}
	summary, errResult := t.applyPlanned(ctx, id, planned)
	if errResult != nil {
		return *errResult, nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: summary}},
	}, nil
}

func (t ApplyPatchTool) parseArgs(id string, args map[string]any) ([]patchOp, *msg.ToolResultMessage) {
	patchText := fmt.Sprint(args["patch"])
	if strings.TrimSpace(patchText) == "" || patchText == "<nil>" {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", "patch is required", "provide a *** Begin Patch block")
		return nil, &res
	}
	if len(patchText) > MaxPatchBytes {
		res := tool.ErrorResult(id, t.Name(), "invalid_args",
			fmt.Sprintf("patch exceeds %d bytes", MaxPatchBytes),
			"split into smaller patches")
		return nil, &res
	}
	ops, err := parseCodexPatch(patchText)
	if err != nil {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", err.Error(), "fix the patch format")
		return nil, &res
	}
	if len(ops) == 0 {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", "patch contains no file operations", "include Add/Update/Delete sections")
		return nil, &res
	}
	if len(ops) > MaxPatchFiles {
		res := tool.ErrorResult(id, t.Name(), "invalid_args",
			fmt.Sprintf("patch touches more than %d files", MaxPatchFiles),
			"split into smaller patches")
		return nil, &res
	}
	return ops, nil
}

func (t ApplyPatchTool) planOps(ctx context.Context, id string, ops []patchOp) ([]plannedPatch, *msg.ToolResultMessage) {
	execEnv := t.executionEnv()
	planned := make([]plannedPatch, 0, len(ops))
	for _, op := range ops {
		item, errResult := t.planOne(ctx, id, execEnv, op)
		if errResult != nil {
			return nil, errResult
		}
		planned = append(planned, item)
	}
	return planned, nil
}

func (t ApplyPatchTool) planOne(ctx context.Context, id string, execEnv env.ExecutionEnv, op patchOp) (plannedPatch, *msg.ToolResultMessage) {
	path := normalizeWorkspacePath(op.Path)
	resolvedResult := t.Workspace.ResolvePath(path)
	if !resolvedResult.IsOk() {
		res := toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue()))
		return plannedPatch{}, &res
	}
	resolved := resolvedResult.Value()
	switch op.Kind {
	case patchAdd:
		if strings.Contains(op.Content, "\x00") {
			res := toolError(id, t.Name(), fmt.Sprintf("binary content not allowed for %s", path))
			return plannedPatch{}, &res
		}
		return plannedPatch{Kind: patchAdd, Path: path, Abs: resolved, Content: []byte(op.Content)}, nil
	case patchDelete:
		return plannedPatch{Kind: patchDelete, Path: path, Abs: resolved}, nil
	case patchUpdate:
		return t.planUpdate(ctx, id, execEnv, path, resolved, op.Hunks)
	default:
		res := toolError(id, t.Name(), fmt.Sprintf("unknown patch operation %q", op.Kind))
		return plannedPatch{}, &res
	}
}

func (t ApplyPatchTool) planUpdate(ctx context.Context, id string, execEnv env.ExecutionEnv, path, resolved string, hunks [][]string) (plannedPatch, *msg.ToolResultMessage) {
	readResult := execEnv.ReadFile(ctx, resolved)
	if !readResult.IsOk() {
		res := toolError(id, t.Name(), env.FormatFileError(readResult.ErrorValue()))
		return plannedPatch{}, &res
	}
	data := readResult.Value()
	if strings.Contains(string(data), "\x00") {
		res := toolError(id, t.Name(), fmt.Sprintf("binary file not allowed for %s", path))
		return plannedPatch{}, &res
	}
	updated, err := applyUpdateHunks(string(data), hunks)
	if err != nil {
		res := toolError(id, t.Name(), fmt.Sprintf("%s: %v", path, err))
		return plannedPatch{}, &res
	}
	if strings.Contains(updated, "\x00") {
		res := toolError(id, t.Name(), fmt.Sprintf("binary content not allowed for %s", path))
		return plannedPatch{}, &res
	}
	return plannedPatch{Kind: patchUpdate, Path: path, Abs: resolved, Content: []byte(updated)}, nil
}

func (t ApplyPatchTool) applyPlanned(ctx context.Context, id string, planned []plannedPatch) (string, *msg.ToolResultMessage) {
	execEnv := t.executionEnv()
	backups, errResult := capturePatchBackups(ctx, id, t.Name(), execEnv, planned)
	if errResult != nil {
		return "", errResult
	}
	summaries := make([]string, 0, len(planned))
	for i, item := range planned {
		line, applyErr := applyOnePlanned(ctx, id, t.Name(), execEnv, item)
		if applyErr != nil {
			_ = restorePatchBackups(ctx, execEnv, backups[:i], planned[:i])
			return "", applyErr
		}
		summaries = append(summaries, line)
	}
	return strings.Join(summaries, "\n"), nil
}

type patchBackup struct {
	Abs     string
	Existed bool
	Content []byte
}

func capturePatchBackups(ctx context.Context, id, toolName string, execEnv env.ExecutionEnv, planned []plannedPatch) ([]patchBackup, *msg.ToolResultMessage) {
	backups := make([]patchBackup, 0, len(planned))
	for _, item := range planned {
		switch item.Kind {
		case patchAdd:
			readResult := execEnv.ReadFile(ctx, item.Abs)
			if readResult.IsOk() {
				backups = append(backups, patchBackup{Abs: item.Abs, Existed: true, Content: append([]byte(nil), readResult.Value()...)})
			} else {
				backups = append(backups, patchBackup{Abs: item.Abs, Existed: false})
			}
		case patchUpdate, patchDelete:
			readResult := execEnv.ReadFile(ctx, item.Abs)
			if !readResult.IsOk() {
				res := toolError(id, toolName, env.FormatFileError(readResult.ErrorValue()))
				return nil, &res
			}
			backups = append(backups, patchBackup{Abs: item.Abs, Existed: true, Content: append([]byte(nil), readResult.Value()...)})
		default:
			backups = append(backups, patchBackup{Abs: item.Abs})
		}
	}
	return backups, nil
}

func restorePatchBackups(ctx context.Context, execEnv env.ExecutionEnv, backups []patchBackup, applied []plannedPatch) error {
	// Restore in reverse order of application.
	for i := len(applied) - 1; i >= 0; i-- {
		item := applied[i]
		backup := backups[i]
		switch item.Kind {
		case patchAdd:
			if backup.Existed {
				_ = execEnv.MkdirAll(ctx, filepath.Dir(backup.Abs), 0o755)
				_ = execEnv.WriteFile(ctx, backup.Abs, backup.Content, 0o644)
			} else {
				_ = execEnv.Remove(ctx, item.Abs)
			}
		case patchUpdate, patchDelete:
			if !backup.Existed {
				continue
			}
			_ = execEnv.MkdirAll(ctx, filepath.Dir(backup.Abs), 0o755)
			_ = execEnv.WriteFile(ctx, backup.Abs, backup.Content, 0o644)
		}
	}
	return nil
}

func applyOnePlanned(ctx context.Context, id, toolName string, execEnv env.ExecutionEnv, item plannedPatch) (string, *msg.ToolResultMessage) {
	switch item.Kind {
	case patchAdd, patchUpdate:
		if mkdirResult := execEnv.MkdirAll(ctx, filepath.Dir(item.Abs), 0o755); !mkdirResult.IsOk() {
			res := toolError(id, toolName, fmt.Sprintf("mkdir: %s", env.FormatFileError(mkdirResult.ErrorValue())))
			return "", &res
		}
		if writeResult := execEnv.WriteFile(ctx, item.Abs, item.Content, 0o644); !writeResult.IsOk() {
			res := toolError(id, toolName, fmt.Sprintf("write %s: %s", item.Path, env.FormatFileError(writeResult.ErrorValue())))
			return "", &res
		}
		action := "updated"
		if item.Kind == patchAdd {
			action = "added"
		}
		return fmt.Sprintf("%s %s (%d bytes)", action, item.Path, len(item.Content)), nil
	case patchDelete:
		if removeResult := execEnv.Remove(ctx, item.Abs); !removeResult.IsOk() {
			res := toolError(id, toolName, fmt.Sprintf("delete %s: %s", item.Path, env.FormatFileError(removeResult.ErrorValue())))
			return "", &res
		}
		return "deleted " + item.Path, nil
	default:
		res := toolError(id, toolName, fmt.Sprintf("unknown planned operation %q", item.Kind))
		return "", &res
	}
}

func (t ApplyPatchTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func splitPatchLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(text, "\n")
}

func applyUpdateHunks(original string, hunks [][]string) (string, error) {
	lines := splitFileLines(original)
	for _, hunk := range hunks {
		before, after, err := hunkBeforeAfter(hunk)
		if err != nil {
			return "", err
		}
		idx := indexSequence(lines, before)
		if idx < 0 {
			return "", fmt.Errorf("hunk context not found")
		}
		next := make([]string, 0, len(lines)-len(before)+len(after))
		next = append(next, lines[:idx]...)
		next = append(next, after...)
		next = append(next, lines[idx+len(before):]...)
		lines = next
	}
	return joinFileLines(lines, strings.HasSuffix(original, "\n")), nil
}

func hunkBeforeAfter(hunk []string) (before, after []string, err error) {
	for _, raw := range hunk {
		if len(raw) == 0 {
			continue
		}
		switch raw[0] {
		case ' ':
			before = append(before, raw[1:])
			after = append(after, raw[1:])
		case '-':
			before = append(before, raw[1:])
		case '+':
			after = append(after, raw[1:])
		default:
			return nil, nil, fmt.Errorf("invalid hunk line %q", raw)
		}
	}
	return before, after, nil
}

func splitFileLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinFileLines(lines []string, trailingNewline bool) string {
	text := strings.Join(lines, "\n")
	if trailingNewline {
		return text + "\n"
	}
	return text
}

func indexSequence(haystack, needle []string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// PatchFilePaths extracts relative file paths from a Codex-style patch for sensors and permissions.
func PatchFilePaths(patch string) []string {
	ops, err := parseCodexPatch(patch)
	if err != nil || len(ops) == 0 {
		return nil
	}
	paths := make([]string, 0, len(ops))
	seen := make(map[string]struct{}, len(ops))
	for _, op := range ops {
		path := normalizeWorkspacePath(op.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

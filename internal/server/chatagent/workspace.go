package chatagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// WorkspaceChoice is one option for the composer workspace picker.
type WorkspaceChoice struct {
	// Value is the relative path under chat_agent.workspace (empty = config root).
	Value string `json:"value"`
	// Label is the display name (config root basename or subdirectory name).
	Label string `json:"label"`
}

// NormalizeWorkspaceRel validates a session workspace relative path.
// Empty means the config root. Non-empty must be a single path segment (no separators, "..", or dotfiles).
func NormalizeWorkspaceRel(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("workspace must be relative: %w", types.ErrInvalidArgument)
	}
	cleaned := filepath.Clean(rel)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid workspace path: %w", types.ErrInvalidArgument)
	}
	if cleaned != filepath.Base(cleaned) {
		return "", fmt.Errorf("workspace must be a top-level directory: %w", types.ErrInvalidArgument)
	}
	if strings.ContainsAny(cleaned, `/\`) {
		return "", fmt.Errorf("workspace must be a top-level directory: %w", types.ErrInvalidArgument)
	}
	if strings.HasPrefix(cleaned, ".") {
		return "", fmt.Errorf("workspace must not be a hidden directory: %w", types.ErrInvalidArgument)
	}
	return cleaned, nil
}

// ConfigWorkspaceRootLabel returns filepath.Base of the configured workspace root.
func ConfigWorkspaceRootLabel() string {
	root := strings.TrimSpace(config.App.ChatAgent.Workspace)
	if root == "" {
		return "workspace"
	}
	base := filepath.Base(root)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "workspace"
	}
	return base
}

// ListWorkspaceChoices returns the config root plus first-level non-hidden subdirectories.
func ListWorkspaceChoices() []WorkspaceChoice {
	rootLabel := ConfigWorkspaceRootLabel()
	out := []WorkspaceChoice{{Value: "", Label: rootLabel}}

	cfgRoot := strings.TrimSpace(config.App.ChatAgent.Workspace)
	if cfgRoot == "" {
		return out
	}
	abs, err := filepath.Abs(cfgRoot)
	if err != nil {
		flog.Warn("[chat-agent] list workspace choices: %v", err)
		return out
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		flog.Warn("[chat-agent] list workspace choices: %v", err)
		return out
	}
	names := make([]string, 0)
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		out = append(out, WorkspaceChoice{Value: name, Label: name})
	}
	return out
}

// ResolveWorkspace builds a coding.Workspace from a relative path under chat_agent.workspace.
func ResolveWorkspace(rel string) (coding.Workspace, error) {
	norm, err := NormalizeWorkspaceRel(rel)
	if err != nil {
		return coding.Workspace{}, err
	}
	cfg := config.App.ChatAgent
	root := strings.TrimSpace(cfg.Workspace)
	if root == "" {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace: %w", err)
	}
	target := absRoot
	if norm != "" {
		target = filepath.Join(absRoot, norm)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return coding.Workspace{}, fmt.Errorf("workspace: %w", err)
	}
	relToRoot, err := filepath.Rel(absRoot, abs)
	if err != nil || strings.HasPrefix(relToRoot, "..") {
		return coding.Workspace{}, fmt.Errorf("workspace escapes chat_agent.workspace: %w", types.ErrInvalidArgument)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			if norm == "" {
				return coding.Workspace{}, fmt.Errorf("chat_agent.workspace: %w", err)
			}
			return coding.Workspace{}, fmt.Errorf("workspace %q not found: %w", norm, types.ErrInvalidArgument)
		}
		return coding.Workspace{}, fmt.Errorf("workspace %q: %w", norm, err)
	}
	if !info.IsDir() {
		if norm == "" {
			return coding.Workspace{}, fmt.Errorf("chat_agent.workspace is not a directory")
		}
		return coding.Workspace{}, fmt.Errorf("workspace %q is not a directory: %w", norm, types.ErrInvalidArgument)
	}

	timeout := cfg.ShellTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxOutput := cfg.MaxToolOutput
	if maxOutput <= 0 {
		maxOutput = 8192
	}
	return coding.Workspace{
		Root:            abs,
		Timeout:         timeout,
		MaxOutput:       maxOutput,
		WebSearchAPIKey: strings.TrimSpace(cfg.WebSearch.APIKey),
	}, nil
}

// SessionWorkspaceRel returns the persisted relative workspace for a session (empty = config root).
func SessionWorkspaceRel(ctx context.Context, sessionID string) (string, error) {
	if store.Database == nil {
		return "", types.ErrUnavailable
	}
	sess, err := store.ChatStoreFromDB().GetChatSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(sess.Workspace), nil
}

// WorkspaceForSession resolves the effective coding.Workspace for a session.
func WorkspaceForSession(ctx context.Context, sessionID string) (coding.Workspace, error) {
	rel, err := SessionWorkspaceRel(ctx, sessionID)
	if err != nil {
		return coding.Workspace{}, err
	}
	return ResolveWorkspace(rel)
}

// ValidateWorkspaceRel normalizes a create-time relative path and checks the directory exists.
func ValidateWorkspaceRel(rel string) (string, error) {
	norm, err := NormalizeWorkspaceRel(rel)
	if err != nil {
		return "", err
	}
	if _, err := ResolveWorkspace(norm); err != nil {
		return "", err
	}
	return norm, nil
}

// ApplyCreateWorkspace validates and persists a create-time workspace relative path.
func ApplyCreateWorkspace(ctx context.Context, sessionID, rel string) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	norm, err := ValidateWorkspaceRel(rel)
	if err != nil {
		return err
	}
	if norm == "" {
		return nil
	}
	if err := store.ChatStoreFromDB().UpdateChatSessionWorkspace(ctx, sessionID, norm); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] set session workspace session=%s: %w", sessionID, err))
		return err
	}
	flog.Debug("[chat-agent] session workspace set session=%s workspace=%s", sessionID, norm)
	return nil
}

// AbortCreatedSession closes a session that failed during create-time setup.
func AbortCreatedSession(ctx context.Context, sessionID string) {
	if store.Database == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	if err := store.ChatStoreFromDB().CloseChatSession(ctx, sessionID); err != nil {
		flog.Warn("[chat-agent] abort created session=%s: %v", sessionID, err)
	}
}

// RejectSettingsWorkspaceField returns ErrInvalidArgument when the settings JSON includes "workspace".
func RejectSettingsWorkspaceField(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := sonic.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("invalid json: %w", types.ErrInvalidArgument)
	}
	if _, ok := raw["workspace"]; ok {
		return fmt.Errorf("workspace cannot be changed after create: %w", types.ErrInvalidArgument)
	}
	return nil
}

package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/clip"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// ModeNormal allows full tool access according to user permissions.
	ModeNormal = string(schema.ChatSessionModeNormal)
	// ModePlan restricts the agent to read-only research tools.
	ModePlan = string(schema.ChatSessionModePlan)
)

// IsReadOnlyTool reports whether name is allowed in plan mode.
func IsReadOnlyTool(name string) bool {
	switch name {
	case "read_file", "web_search", "web_fetch", "read_skill", "list_dir", "glob_files", "grep_files", listScheduleToolName, clip.GetToolName:
		return true
	default:
		return false
	}
}

// ReadOnlyToolNames returns the active tool set for plan mode.
func ReadOnlyToolNames() []string {
	return []string{
		"list_dir", "glob_files", "grep_files", "read_file",
		"web_search", "web_fetch", "read_skill", listScheduleToolName, updateMemoryToolName,
		clip.GetToolName,
	}
}

// ValidSessionMode reports whether mode is a supported session mode value.
func ValidSessionMode(mode string) bool {
	switch mode {
	case ModeNormal, ModePlan:
		return true
	default:
		return false
	}
}

// LoadSessionMode reads the persisted mode for one session, defaulting to normal.
func LoadSessionMode(ctx context.Context, sessionID string) string {
	if store.Database == nil || sessionID == "" {
		return ModeNormal
	}
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		flog.Debug("[chat-agent] load session mode session=%s: %v", sessionID, err)
		return ModeNormal
	}
	mode := strings.TrimSpace(sess.Mode)
	if ValidSessionMode(mode) {
		return mode
	}
	return ModeNormal
}

// SetSessionMode persists a session mode toggle.
func SetSessionMode(ctx context.Context, sessionID, mode string) error {
	if !ValidSessionMode(mode) {
		return fmt.Errorf("invalid session mode: %q", mode)
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	return store.Database.UpdateChatSessionMode(ctx, sessionID, mode)
}

// NotifySessionModeChange publishes a mode_change event to session SSE subscribers.
func NotifySessionModeChange(sessionID, mode string) {
	if sessionID == "" {
		return
	}
	PublishSessionEvent(sessionID, StreamEvent{
		Type: EventTypeModeChange,
		Mode: mode,
	})
}

// SetSessionModeAndNotify persists mode and notifies connected SSE clients.
func SetSessionModeAndNotify(ctx context.Context, sessionID, mode string) error {
	if err := SetSessionMode(ctx, sessionID, mode); err != nil {
		return err
	}
	NotifySessionModeChange(sessionID, mode)
	return nil
}

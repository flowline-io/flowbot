package chatagent

import (
	"context"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const sessionPreviewMaxLen = 96

// UpdateSessionPreview persists a truncated last-message snippet for session lists.
func UpdateSessionPreview(ctx context.Context, sessionID, text string) {
	if store.Database == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	preview := truncateSessionPreview(text, sessionPreviewMaxLen)
	if err := store.Database.UpdateChatSessionPreview(ctx, sessionID, preview); err != nil {
		flog.Warn("[chat-agent] update session preview session=%s: %v", sessionID, err)
	}
}

func truncateSessionPreview(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return ""
	}
	if limit <= 0 {
		limit = sessionPreviewMaxLen
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

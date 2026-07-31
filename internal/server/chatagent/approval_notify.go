package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkgnotify "github.com/flowline-io/flowbot/pkg/notify"
)

const agentApprovalDeepLinkPrefix = "/service/web/agents/"

func (g *ConfirmGate) notifyApprovalPending(confirmID, summary string) {
	if g == nil || g.sessionID == "" || confirmID == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		uid, err := SessionOwnerUID(ctx, g.sessionID)
		if err != nil || uid.IsZero() {
			flog.Warn("[chatagent] approval notify: resolve uid for %s: %v", g.sessionID, err)
			return
		}
		escalate := pkgnotify.EscalateAfter()
		if g.timeout > 0 && g.timeout < escalate {
			escalate = g.timeout
		}
		source := sessionInboxSourceLabel(ctx, g.sessionID)
		payload := map[string]any{
			pkgnotify.PayloadKeySummary:       fmt.Sprintf("Needs approval · From %s", source),
			pkgnotify.PayloadKeyURL:           agentApprovalDeepLinkPrefix + g.sessionID,
			pkgnotify.PayloadKeyCorrelationID: confirmID,
			pkgnotify.PayloadKeyEscalateAfter: escalate.String(),
			pkgnotify.PayloadKeyTitle:         summary,
			"session_id":                      g.sessionID,
			"source":                          source,
		}
		channels := pkgnotify.DefaultInboxChannels(ctx)
		if err := pkgnotify.GatewaySend(ctx, uid, pkgnotify.AgentApprovalTemplateID, channels, payload); err != nil {
			if !pkgnotify.WarnSkipNoDefault(err, "approval inbox") {
				flog.Warn("[chatagent] approval notify send: %v", err)
			}
		}
	}()
}

func sessionInboxSourceLabel(ctx context.Context, sessionID string) string {
	if store.Database == nil {
		return truncateInboxLabel(sessionID, 12)
	}
	row, err := store.ChatStoreFromDB().GetChatSession(ctx, sessionID)
	if err != nil || row == nil {
		return truncateInboxLabel(sessionID, 12)
	}
	title := strings.TrimSpace(row.Title)
	if title != "" {
		return truncateInboxLabel(title, 48)
	}
	return truncateInboxLabel(sessionID, 12)
}

func truncateInboxLabel(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" || maxRunes <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if maxRunes <= 1 {
		return string(runes[:1])
	}
	return string(runes[:maxRunes-1]) + "…"
}

func markApprovalInboxRead(sessionID, confirmID string) {
	if sessionID == "" || confirmID == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		uid, err := SessionOwnerUID(ctx, sessionID)
		if err != nil || uid.IsZero() {
			return
		}
		ns := pkgnotify.GetNotifyStore()
		if ns == nil {
			return
		}
		if err := ns.MarkReadByCorrelation(ctx, uid.String(), confirmID); err != nil {
			flog.Warn("[chatagent] mark approval inbox read: %v", err)
		}
	}()
}

// MarkApprovalInboxReadForSession marks inbox items for the pending confirm when the session page opens.
func MarkApprovalInboxReadForSession(sessionID string) {
	if sessionID == "" || scheduledRunService == nil {
		return
	}
	raw, ok := scheduledRunService.sessionConfirmGates.Load(sessionID)
	if !ok {
		return
	}
	gate, ok := raw.(*ConfirmGate)
	if !ok || gate == nil || !gate.IsWaiting() {
		return
	}
	ev, ok := gate.PendingEvent()
	if !ok || ev.ID == "" {
		return
	}
	markApprovalInboxRead(sessionID, ev.ID)
}

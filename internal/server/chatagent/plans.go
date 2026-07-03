package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// maybePersistPlan stores the assistant reply as a plan document when session mode is plan.
func maybePersistPlan(ctx context.Context, sessionID, reply string) (planID string, title string, ok bool) {
	reply = strings.TrimSpace(reply)
	if reply == "" || LoadSessionMode(ctx, sessionID) != ModePlan {
		return "", "", false
	}
	if store.Database == nil {
		flog.Warn("[chat-agent] plan store unavailable session=%s", sessionID)
		return "", "", false
	}
	title = derivePlanTitle(reply)
	planID = types.Id()
	now := time.Now().UTC()
	row := &gen.AgentPlan{
		Flag:      planID,
		SessionID: sessionID,
		Title:     title,
		Content:   reply,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Database.CreateAgentPlan(ctx, row); err != nil {
		flog.Warn("[chat-agent] persist plan session=%s: %v", sessionID, err)
		return "", "", false
	}
	flog.Debug("[chat-agent] persisted plan session=%s plan=%s title=%q", sessionID, planID, title)
	return planID, title, true
}

// derivePlanTitle extracts a short title from plan markdown text.
func derivePlanTitle(reply string) string {
	for line := range strings.SplitSeq(reply, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 80 {
				return line[:80]
			}
			return line
		}
	}
	return "Plan"
}

// FormatPlanResourceRef builds a plan resource reference for SSE and clients.
func FormatPlanResourceRef(planID, title string) ResourceRef {
	return ResourceRef{
		URI:   PlanLocationPrefix + planID,
		Kind:  "plan",
		Title: title,
	}
}

// AppendPlanLinkFooter appends a markdown plan link to the assistant reply.
func AppendPlanLinkFooter(reply, planID, title string) string {
	reply = strings.TrimRight(reply, "\n")
	return reply + fmt.Sprintf("\n\n---\nPlan saved: [%s](%s%s)", title, PlanLocationPrefix, planID)
}

// PlanSummary is a lightweight plan listing item for HTTP clients.
type PlanSummary struct {
	PlanID    string    `json:"plan_id"`
	URI       string    `json:"uri"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// ListPlanSummaries returns plan metadata for one session.
func ListPlanSummaries(ctx context.Context, sessionID string) ([]PlanSummary, error) {
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	rows, err := store.Database.ListAgentPlansBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]PlanSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, PlanSummary{
			PlanID:    row.Flag,
			URI:       PlanLocationPrefix + row.Flag,
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

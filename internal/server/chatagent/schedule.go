package chatagent

import (
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cronutil"
)

const (
	scheduleToolName       = "schedule_task"
	updateScheduleToolName = "update_scheduled_task"
	listScheduleToolName   = "list_scheduled_tasks"
	cancelScheduleToolName = "cancel_scheduled_task"

	onceGraceWindow = 5 * time.Minute
)

// ScheduledDelivery captures where to push task results.
type ScheduledDelivery struct {
	Platform   string `json:"platform,omitempty"`
	Topic      string `json:"topic,omitempty"`
	PlatformID int64  `json:"platform_id,omitempty"`
}

// ValidScheduleKind reports whether kind is supported.
func ValidScheduleKind(kind string) bool {
	switch kind {
	case string(schema.ChatScheduledTaskKindCron), string(schema.ChatScheduledTaskKindOnce):
		return true
	default:
		return false
	}
}

// ValidScheduledTaskState reports whether state is a known task lifecycle value.
func ValidScheduledTaskState(state string) bool {
	switch state {
	case string(schema.ChatScheduledTaskStateActive),
		string(schema.ChatScheduledTaskStatePaused),
		string(schema.ChatScheduledTaskStateCancelled),
		string(schema.ChatScheduledTaskStateCompleted),
		string(schema.ChatScheduledTaskStateFailed),
		string(schema.ChatScheduledTaskStateMissed):
		return true
	default:
		return false
	}
}

// EditableScheduledTaskState reports whether schedule fields may be changed.
func EditableScheduledTaskState(state string) bool {
	return state == string(schema.ChatScheduledTaskStateActive) ||
		state == string(schema.ChatScheduledTaskStatePaused)
}

// IsScheduleWriteTool reports tools blocked in plan mode.
func IsScheduleWriteTool(name string) bool {
	switch name {
	case scheduleToolName, updateScheduleToolName, cancelScheduleToolName:
		return true
	default:
		return false
	}
}

// ParseRunAt parses an ISO8601 UTC timestamp for one-shot tasks.
func ParseRunAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errRunAtRequired
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errInvalidRunAt
	}
	return t.UTC(), nil
}

// ValidateScheduleInput validates create/update schedule fields.
func ValidateScheduleInput(kind, cronExpr string, runAt *time.Time) error {
	switch kind {
	case string(schema.ChatScheduledTaskKindCron):
		if strings.TrimSpace(cronExpr) == "" {
			return errCronRequired
		}
		if runAt != nil {
			return errCronRunAtConflict
		}
		return cronutil.ValidateExpr(cronExpr)
	case string(schema.ChatScheduledTaskKindOnce):
		if strings.TrimSpace(cronExpr) != "" {
			return errCronRunAtConflict
		}
		if runAt == nil || runAt.IsZero() {
			return errRunAtRequired
		}
		if !runAt.After(time.Now().UTC()) {
			return errRunAtPast
		}
		return nil
	default:
		return errInvalidScheduleKind
	}
}

func promptSummary(prompt string, limit int) string {
	prompt = strings.TrimSpace(prompt)
	if limit <= 0 {
		limit = 80
	}
	if len(prompt) <= limit {
		return prompt
	}
	return prompt[:limit] + "..."
}

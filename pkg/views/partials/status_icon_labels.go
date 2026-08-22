package partials

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

func sessionStateLabel(ctx context.Context, state string) string {
	switch state {
	case "Active":
		return i18n.T(ctx, "status.session.active")
	case "Closed":
		return i18n.T(ctx, "status.session.closed")
	default:
		return state
	}
}

func scheduledTaskStateLabel(ctx context.Context, state string) string {
	switch state {
	case "active":
		return i18n.T(ctx, "status.scheduled_task.active")
	case "paused":
		return i18n.T(ctx, "status.scheduled_task.paused")
	case "completed":
		return i18n.T(ctx, "status.scheduled_task.completed")
	case "failed":
		return i18n.T(ctx, "status.scheduled_task.failed")
	case "cancelled":
		return i18n.T(ctx, "status.scheduled_task.cancelled")
	case "missed":
		return i18n.T(ctx, "status.scheduled_task.missed")
	default:
		return state
	}
}

func scheduledTaskRunStateLabel(ctx context.Context, state string) string {
	switch state {
	case "completed":
		return i18n.T(ctx, "status.scheduled_task_run.completed")
	case "running":
		return i18n.T(ctx, "status.scheduled_task_run.running")
	case "failed":
		return i18n.T(ctx, "status.scheduled_task_run.failed")
	default:
		return state
	}
}

func pipelineRuntimeLabel(ctx context.Context, enabled bool) string {
	if enabled {
		return i18n.T(ctx, "status.pipeline.active")
	}
	return i18n.T(ctx, "status.pipeline.paused")
}

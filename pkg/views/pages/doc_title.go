package pages

import (
	"context"
	"strings"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// DocTitleFlowbot returns a localized browser title: "{navLabel} — Flowbot".
func DocTitleFlowbot(ctx context.Context, navKey string) string {
	return i18n.T(ctx, navKey) + i18n.T(ctx, "common.doc_title_flowbot")
}

// DocTitlePage returns a localized browser title from a page.* key plus Flowbot suffix.
func DocTitlePage(ctx context.Context, pageKey string) string {
	return i18n.T(ctx, pageKey) + i18n.T(ctx, "common.doc_title_flowbot")
}

// DocTitleLife returns a localized browser title from a life page key plus Life suffix.
func DocTitleLife(ctx context.Context, pageKey string) string {
	return i18n.T(ctx, pageKey) + i18n.T(ctx, "common.doc_title_life")
}

// DocTitleNamed returns a localized title with template data plus Flowbot suffix.
func DocTitleNamed(ctx context.Context, titleKey string, data map[string]any) string {
	return i18n.TData(ctx, titleKey, data) + i18n.T(ctx, "common.doc_title_flowbot")
}

// DocTitleLiteral returns a dynamic name plus Flowbot suffix.
func DocTitleLiteral(ctx context.Context, name string) string {
	return name + i18n.T(ctx, "common.doc_title_flowbot")
}

// DocTitleAgentSession returns a localized browser title for an agent session detail page.
func DocTitleAgentSession(ctx context.Context, session model.AgentSession) string {
	if name := strings.TrimSpace(session.Title); name != "" {
		return DocTitleLiteral(ctx, name)
	}
	return DocTitleLiteral(ctx, session.Flag)
}

// DocTitleAgentScheduledTask returns a localized browser title for a scheduled task detail page.
func DocTitleAgentScheduledTask(ctx context.Context, task model.AgentScheduledTask) string {
	if name := strings.TrimSpace(task.Name); name != "" {
		return DocTitleLiteral(ctx, name)
	}
	return DocTitleLiteral(ctx, task.TaskID)
}

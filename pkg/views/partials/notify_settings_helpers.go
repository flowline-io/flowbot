package partials

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func notifyChannelRowID(item model.NotifyChannel) string {
	return fmt.Sprintf("notify-channel-%d", item.ID)
}

func notifyChannelEditURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notifications/channels/%d/edit", item.ID)
}

func notifyChannelDeleteURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notifications/channels/%d", item.ID)
}

func notifyChannelUpdateURL(item model.NotifyChannel) string {
	return notifyChannelDeleteURL(item)
}

func notifyChannelTestURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notifications/channels/%d/test", item.ID)
}

func notifyRuleRowID(item model.NotifyRule) string {
	return fmt.Sprintf("notify-rule-%d", item.ID)
}

func notifyRuleEditURL(item model.NotifyRule) string {
	return fmt.Sprintf("/service/web/notifications/rules/%d/edit", item.ID)
}

func notifyRuleDeleteURL(item model.NotifyRule) string {
	return fmt.Sprintf("/service/web/notifications/rules/%d", item.ID)
}

func notifyRuleUpdateURL(item model.NotifyRule) string {
	return notifyRuleDeleteURL(item)
}

func notifyChannelFormID(item model.NotifyChannel, isNew bool) string {
	if isNew {
		return "notify-channel-form-new"
	}
	return "notify-channel-form-" + notifyChannelRowID(item)
}

func notifyRuleFormID(item model.NotifyRule, isNew bool) string {
	if isNew {
		return "notify-rule-form-new"
	}
	return "notify-rule-form-" + notifyRuleRowID(item)
}

func notifyChannelCancelURL() string {
	return "/service/web/notifications/channels/list"
}

func notifyRuleCancelURL() string {
	return "/service/web/notifications/rules/list"
}

func notifyTemplateRowID(item model.NotifyTemplate) string {
	return fmt.Sprintf("notify-template-%d", item.ID)
}

func notifyTemplateEditURL(item model.NotifyTemplate) string {
	return fmt.Sprintf("/service/web/notifications/templates/%d/edit", item.ID)
}

func notifyTemplateDeleteURL(item model.NotifyTemplate) string {
	return fmt.Sprintf("/service/web/notifications/templates/%d", item.ID)
}

func notifyTemplateUpdateURL(item model.NotifyTemplate) string {
	return notifyTemplateDeleteURL(item)
}

func notifyTemplateFormID(item model.NotifyTemplate, isNew bool) string {
	if isNew {
		return "notify-template-form-new"
	}
	return "notify-template-form-" + notifyTemplateRowID(item)
}

func notifyTemplateCancelURL() string {
	return "/service/web/notifications/templates/list"
}

func notifyTemplateOverrideCount(item model.NotifyTemplate) int {
	if item.OverridesJSON == "" || item.OverridesJSON == "[]" {
		return 0
	}
	var overrides []any
	if sonic.Unmarshal([]byte(item.OverridesJSON), &overrides) != nil {
		return 0
	}
	return len(overrides)
}

func actionBadgeClass(action string) string {
	switch action {
	case "throttle":
		return "badge badge-warning"
	case "aggregate":
		return "badge badge-info"
	case "mute":
		return "badge badge-ghost"
	case "drop":
		return "badge badge-error"
	default:
		return "badge"
	}
}

func enabledBadgeClass(enabled bool) string {
	if enabled {
		return "badge badge-success"
	}
	return "badge badge-ghost"
}

func enabledText(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func escapePath(s string) string {
	return url.PathEscape(s)
}

func hasTemplateForRule(item model.NotifyRule, templateIDs []string) bool {
	if item.ParamsJSON == "" {
		return true
	}
	var params map[string]any
	if sonic.Unmarshal([]byte(item.ParamsJSON), &params) != nil {
		return true // can't parse - don't flag as stale
	}
	tid, ok := params["digest_template_id"].(string)
	if !ok || tid == "" {
		return true
	}
	return slices.Contains(templateIDs, tid)
}

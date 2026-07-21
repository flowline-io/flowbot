package partials

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

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

func notifyChannelDefaultURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notifications/channels/%d/default", item.ID)
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

func notifyTemplateDefaultURL(item model.NotifyTemplate) string {
	return fmt.Sprintf("/service/web/notifications/templates/%d/default", item.ID)
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
		return "flowbot-chip flowbot-chip-warning"
	case "aggregate":
		return "flowbot-chip flowbot-chip-primary"
	case "mute":
		return "flowbot-chip flowbot-chip-muted"
	case "drop":
		return "flowbot-chip flowbot-chip-error"
	default:
		return "flowbot-chip flowbot-chip-muted"
	}
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

// ruleActionSummary returns a short human-readable summary of action parameters for list display.
func ruleActionSummary(item model.NotifyRule) string {
	switch item.Action {
	case "mute":
		return muteRuleActionSummary(item.Condition)
	case "throttle":
		return throttleRuleActionSummary(item.ParamsJSON)
	case "aggregate":
		return aggregateRuleActionSummary(item.ParamsJSON)
	default:
		return ""
	}
}

// muteRuleActionSummary summarizes a mute rule's condition expression.
func muteRuleActionSummary(condition string) string {
	if condition == "" {
		return ""
	}
	return truncateString(condition, 48)
}

// throttleRuleActionSummary summarizes throttle window and limit params.
func throttleRuleActionSummary(paramsJSON string) string {
	p := parseRuleParamsFields(paramsJSON)
	return joinRuleSummaryParts(
		labeledRuleSummaryPart("window", p.Window),
		labeledRuleSummaryPart("limit", p.Limit),
	)
}

// aggregateRuleActionSummary summarizes aggregate window, digest, and delay params.
func aggregateRuleActionSummary(paramsJSON string) string {
	p := parseRuleParamsFields(paramsJSON)
	parts := []string{
		labeledRuleSummaryPart("window", p.Window),
		labeledRuleSummaryPart("digest", p.DigestTemplateID),
	}
	if p.DelayedSend {
		parts = append(parts, "delayed")
	}
	return joinRuleSummaryParts(parts...)
}

// labeledRuleSummaryPart formats "label value" when value is non-empty.
func labeledRuleSummaryPart(label, value string) string {
	if value == "" {
		return ""
	}
	return label + " " + value
}

// joinRuleSummaryParts joins non-empty summary fragments with a middle dot separator.
func joinRuleSummaryParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}

// ruleParamsFields holds structured rule parameter fields for the graphical form.
type ruleParamsFields struct {
	Window           string
	Limit            string
	DigestTemplateID string
	DelayedSend      bool
}

// parseRuleParamsFields extracts display fields from a rule's ParamsJSON.
func parseRuleParamsFields(paramsJSON string) ruleParamsFields {
	out := ruleParamsFields{}
	if paramsJSON == "" {
		return out
	}
	var params map[string]any
	if err := sonic.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return out
	}
	if w, ok := params["window"].(string); ok && w != "" {
		out.Window = w
	}
	switch v := params["limit"].(type) {
	case float64:
		out.Limit = fmt.Sprintf("%d", int(v))
	case int:
		out.Limit = fmt.Sprintf("%d", v)
	case string:
		if v != "" {
			out.Limit = v
		}
	}
	if tid, ok := params["digest_template_id"].(string); ok {
		out.DigestTemplateID = tid
	}
	if d, ok := params["delayed_send"].(bool); ok {
		out.DelayedSend = d
	}
	return out
}

// ruleFormWindow returns the window value for the rule form, with a sensible default.
func ruleFormWindow(p ruleParamsFields) string {
	if p.Window == "" {
		return "5m"
	}
	return p.Window
}

// ruleFormLimit returns the limit value for the rule form, with a sensible default.
func ruleFormLimit(p ruleParamsFields) string {
	if p.Limit == "" {
		return "1"
	}
	return p.Limit
}

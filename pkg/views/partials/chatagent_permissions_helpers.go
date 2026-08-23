package partials

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// PermissionKeyMeta describes one editable permission key for the web form.
type PermissionKeyMeta struct {
	Key              string
	Label            string
	Description      string
	SupportsPatterns bool
	DisallowAllow    bool
}

// PermissionPatternRow is one pattern rule row in the permissions form.
type PermissionPatternRow struct {
	Pattern string
	Action  string
}

// PermissionFormField is the view model for one permission key row.
type PermissionFormField struct {
	Meta           PermissionKeyMeta
	DefaultSummary string
	SelectedAction string
	PatternRows    []PermissionPatternRow
	IsOverridden   bool
}

// PermissionFormPageData is the full page model for the permissions editor.
type PermissionFormPageData struct {
	Fields            []PermissionFormField
	UserJSON          string
	Errors            map[string]string
	ApprovalMode      string
	ApprovalServerDef string
	ServerDefaults    ServerDefaultsFormData
}

// ServerDefaultsFormData is the server-wide model defaults section on the permissions page.
type ServerDefaultsFormData struct {
	YAMLChatModel         string
	YAMLToolModel         string
	InheritValue          string
	ToolNoneValue         string
	SelectableModels      []SelectableModelOption
	SelectedChatModel     string
	SelectedToolModel     string
	SelectedThinkingLevel string
	ChatModelOverridden   bool
	ToolModelOverridden   bool
	ThinkingOverridden    bool
}

// permissionKeyCatalog is the ordered list of user-editable permission keys.
var permissionKeyCatalog = []PermissionKeyMeta{
	{Key: "websearch", Label: "Web Search", Description: "Controls web_search tool access.", DisallowAllow: false},
	{Key: "skill", Label: "Skills", Description: "Controls read_skill tool access.", DisallowAllow: false},
	{Key: "knowledge", Label: "Knowledge", Description: "Controls search_knowledge and get_knowledge tool access.", DisallowAllow: false},
	{Key: "delegate", Label: "Delegate", Description: "Controls task (subagent) delegation.", DisallowAllow: true},
	{Key: "schedule", Label: "Schedule Write", Description: "Controls schedule_task, update_scheduled_task, and cancel_scheduled_task.", DisallowAllow: true},
	{Key: "schedule_read", Label: "Schedule Read", Description: "Controls list_scheduled_tasks.", DisallowAllow: false},
	{Key: "doom_loop", Label: "Doom Loop", Description: "Controls loop-detection critical hits (ask/deny/allow). Global and post-compaction hard-stops ignore this key.", DisallowAllow: false},
	{Key: "read", Label: "Read Files", Description: "Controls read_file access by file path pattern.", SupportsPatterns: true, DisallowAllow: true},
	{Key: "edit", Label: "Edit Files", Description: "Controls write_file access by file path pattern.", SupportsPatterns: true, DisallowAllow: true},
	{Key: "bash", Label: "Shell / Code", Description: "Controls run_terminal and run_code by command pattern.", SupportsPatterns: true, DisallowAllow: true},
	{Key: permission.KeyExternalDirectory, Label: "External Paths", Description: "Controls access to paths outside the workspace.", SupportsPatterns: true, DisallowAllow: true},
}

// BuildPermissionFormFields builds form rows from a permissions API view.
func BuildPermissionFormFields(ctx context.Context, view model.PermissionsView) []PermissionFormField {
	fields := make([]PermissionFormField, 0, len(permissionKeyCatalog))
	for _, meta := range permissionKeyCatalog {
		meta = localizePermissionMeta(ctx, meta)
		field := PermissionFormField{
			Meta:           meta,
			DefaultSummary: FormatRuleSetSummary(ctx, view.Defaults[meta.Key]),
			SelectedAction: permission.FormActionInherit,
		}
		userRS, hasUser := view.User[meta.Key]
		if meta.SupportsPatterns {
			if hasUser && len(userRS.Patterns) > 0 {
				field.IsOverridden = true
				field.PatternRows = patternRowsFromRuleSet(userRS)
			}
			fields = append(fields, field)
			continue
		}
		if hasUser && userRS.Default.Valid() {
			field.IsOverridden = true
			field.SelectedAction = string(userRS.Default)
		}
		fields = append(fields, field)
	}
	return fields
}

func patternRowsFromRuleSet(rs permission.RuleSet) []PermissionPatternRow {
	rows := make([]PermissionPatternRow, 0, len(rs.Patterns))
	for _, rule := range rs.Patterns {
		rows = append(rows, PermissionPatternRow{
			Pattern: rule.Pattern,
			Action:  string(rule.Action),
		})
	}
	return rows
}

// FormatRuleSetSummary returns a compact human-readable summary of one rule set.
func FormatRuleSetSummary(ctx context.Context, rs permission.RuleSet) string {
	if len(rs.Patterns) == 0 {
		if rs.Default.Valid() {
			return PermissionActionLabel(ctx, string(rs.Default))
		}
		return PermissionActionLabel(ctx, "ask")
	}
	parts := make([]string, 0, len(rs.Patterns))
	for _, rule := range rs.Patterns {
		parts = append(parts, fmt.Sprintf("%s → %s", rule.Pattern, PermissionActionLabel(ctx, string(rule.Action))))
	}
	return strings.Join(parts, ", ")
}

// PermissionActionLabel returns a localized permission action label.
func PermissionActionLabel(ctx context.Context, action string) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		return ""
	}
	return i18n.TDefault(ctx, "permissions.action."+action, action)
}

// PermissionApprovalModeLabel returns a localized approval mode label.
func PermissionApprovalModeLabel(ctx context.Context, mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return ""
	}
	return i18n.TDefault(ctx, "permissions.approval_mode."+mode, mode)
}

func localizePermissionMeta(ctx context.Context, meta PermissionKeyMeta) PermissionKeyMeta {
	return PermissionKeyMeta{
		Key:              meta.Key,
		Label:            i18n.TDefault(ctx, "permissions.key."+meta.Key+".label", meta.Label),
		Description:      i18n.TDefault(ctx, "permissions.key."+meta.Key+".description", meta.Description),
		SupportsPatterns: meta.SupportsPatterns,
		DisallowAllow:    meta.DisallowAllow,
	}
}

// PermissionFieldError returns a validation error for one permission field.
func PermissionFieldError(errors map[string]string, key string) string {
	if errors == nil {
		return ""
	}
	return errors["perm."+key]
}

// PermissionPatternError returns a validation error for one pattern row.
func PermissionPatternError(errors map[string]string, key string, idx int) string {
	if errors == nil {
		return ""
	}
	return errors[fmt.Sprintf("perm.%s.patterns.%d.pattern", key, idx)]
}

// PermissionPatternActionError returns a validation error for one pattern action.
func PermissionPatternActionError(errors map[string]string, key string, idx int) string {
	if errors == nil {
		return ""
	}
	return errors[fmt.Sprintf("perm.%s.patterns.%d.action", key, idx)]
}

// PermissionPatternErrorClass returns a border class when a pattern field has an error.
func PermissionPatternErrorClass(errors map[string]string, key string, idx int) string {
	if PermissionPatternError(errors, key, idx) != "" {
		return "border-red-500"
	}
	return ""
}

// PermissionPatternActionErrorClass returns a border class when a pattern action has an error.
func PermissionPatternActionErrorClass(errors map[string]string, key string, idx int) string {
	if PermissionPatternActionError(errors, key, idx) != "" {
		return "border-red-500"
	}
	return ""
}

// ApplySubmittedPermissionForm overlays submitted form values onto rendered fields.
func ApplySubmittedPermissionForm(fields []PermissionFormField, form permission.FormValues) []PermissionFormField {
	out := make([]PermissionFormField, len(fields))
	copy(out, fields)
	for i := range out {
		key := out[i].Meta.Key
		if out[i].Meta.SupportsPatterns {
			if rows, ok := form.Patterns[key]; ok {
				out[i].PatternRows = make([]PermissionPatternRow, len(rows))
				for j, row := range rows {
					out[i].PatternRows[j] = PermissionPatternRow{
						Pattern: row.Pattern,
						Action:  row.Action,
					}
				}
				out[i].IsOverridden = len(out[i].PatternRows) > 0
			}
			continue
		}
		if action, ok := form.Simple[key]; ok && action != "" {
			out[i].SelectedAction = action
			out[i].IsOverridden = action != permission.FormActionInherit
		}
	}
	return out
}

// ServerDefaultInheritLabel formats the inherit option for one YAML-backed field.
func ServerDefaultInheritLabel(ctx context.Context, yamlValue string) string {
	display := strings.TrimSpace(yamlValue)
	if display == "" {
		return i18n.T(ctx, "permissions.server_defaults.inherit_empty")
	}
	return i18n.TData(ctx, "permissions.server_defaults.inherit_yaml", map[string]any{"Value": display})
}

// ServerDefaultToolInheritLabel formats the inherit option for tool_model.
func ServerDefaultToolInheritLabel(ctx context.Context, yamlTool string) string {
	if strings.TrimSpace(yamlTool) == "" {
		return i18n.T(ctx, "permissions.server_defaults.tool_inherit_single")
	}
	return i18n.TData(ctx, "permissions.server_defaults.inherit_yaml", map[string]any{"Value": yamlTool})
}

// ServerDefaultThinkingInheritLabel formats the inherit option for thinking_level.
func ServerDefaultThinkingInheritLabel(ctx context.Context) string {
	return i18n.T(ctx, "permissions.server_defaults.thinking_inherit")
}

// ApplySubmittedServerDefaults overlays submitted form values onto server defaults data.
func ApplySubmittedServerDefaults(data ServerDefaultsFormData, inheritValue, chatModel, toolModel, thinking string) ServerDefaultsFormData {
	if chat := strings.TrimSpace(chatModel); chat != "" {
		data.SelectedChatModel = chat
		data.ChatModelOverridden = chat != inheritValue
	}
	if tool := strings.TrimSpace(toolModel); tool != "" {
		data.SelectedToolModel = tool
		data.ToolModelOverridden = tool != inheritValue
	}
	if thinkingLevel := strings.TrimSpace(thinking); thinkingLevel != "" {
		data.SelectedThinkingLevel = thinkingLevel
		data.ThinkingOverridden = thinkingLevel != inheritValue
	}
	return data
}

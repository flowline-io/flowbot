package chatagent

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/subagent"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Subagent is a prompt-visible subagent definition loaded from storage.
type Subagent struct {
	Flag         string
	Name         string
	Description  string
	SystemPrompt string
	Tools        []string
	Skills       []string
	Model        string
}

// LoadSubagentsFromStore loads enabled subagent definitions from the database.
func LoadSubagentsFromStore(ctx context.Context) ([]Subagent, error) {
	if store.Database == nil {
		return nil, nil
	}
	rows, err := store.Database.ListAgentSubagents(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("load agent subagents: %w", err)
	}
	subagents := make([]Subagent, 0, len(rows))
	for _, row := range rows {
		subagents = append(subagents, subagentFromRow(row))
	}
	flog.Debug("[chat-agent] loaded %d subagents from store", len(subagents))
	return subagents, nil
}

// GetSubagentDefinition loads one enabled subagent by name as a runnable definition.
func GetSubagentDefinition(ctx context.Context, name string) (subagent.Definition, error) {
	if store.Database == nil {
		return subagent.Definition{}, fmt.Errorf("subagent store unavailable")
	}
	row, err := store.Database.GetAgentSubagentByName(ctx, name)
	if err != nil {
		return subagent.Definition{}, err
	}
	return subagent.Definition{
		Name:         row.Name,
		Description:  row.Description,
		SystemPrompt: row.SystemPrompt,
		Tools:        append([]string(nil), row.Tools...),
		Skills:       append([]string(nil), row.Skills...),
		Model:        row.Model,
	}, nil
}

func subagentFromRow(row *gen.AgentSubagent) Subagent {
	return Subagent{
		Flag:         row.Flag,
		Name:         row.Name,
		Description:  row.Description,
		SystemPrompt: row.SystemPrompt,
		Tools:        append([]string(nil), row.Tools...),
		Skills:       append([]string(nil), row.Skills...),
		Model:        row.Model,
	}
}

// LegacyBuiltinSystemPrompts maps builtin flag -> known pre-rewrite system prompts.
// Rows still carrying these values are migrated once; customized rows are left alone.
var LegacyBuiltinSystemPrompts = map[string][]string{
	"general": {
		"You are a general-purpose subagent. Complete the delegated task end to end using the available " +
			"tools, including reading and modifying project files when needed. Work in isolation, make reasonable " +
			"assumptions, and return a concise, self-contained summary of what you found or did. Do not ask " +
			"follow-up questions.",
	},
	"explore": {
		"You are an explore subagent operating in read-only mode. Use list_dir, glob_files, grep_files, read_file, web_search, and web_fetch to " +
			"navigate the codebase, locate where features are implemented, and explain how complex logic works. " +
			"Never modify files or access write-capable tools. Return a concise, self-contained summary with " +
			"file paths and relevant excerpts.",
	},
	"scout": {
		"You are a scout subagent focused on external information retrieval. Use web_search and web_fetch to find " +
			"official documentation, GitHub repositories, and current API usage. Use run_terminal to clone or " +
			"fetch external dependencies when needed, and read_file to inspect retrieved content. Cross-reference " +
			"sources, verify facts against the latest docs, and return a concise, self-contained summary with " +
			"links and actionable findings.",
	},
}

// LegacyBuiltinDescriptions maps builtin flag -> known pre-rewrite descriptions.
var LegacyBuiltinDescriptions = map[string][]string{
	"general": {"General-purpose subagent for complex research and multi-step tasks; can read and modify project files"},
	"explore": {"Fast read-only subagent for codebase navigation, locating implementations, and understanding complex logic"},
	"scout":   {"Research subagent for external docs, APIs, GitHub, and dependency investigation beyond the training cutoff"},
}

// BuiltinSubagentFields holds prompt fields considered for builtin migration.
type BuiltinSubagentFields struct {
	Source       string
	Flag         string
	SystemPrompt string
	Description  string
}

// MigrateBuiltinSubagentFields returns updated prompt fields when the row still
// carries a known legacy builtin prompt or description. Unchanged fields are returned as-is.
func MigrateBuiltinSubagentFields(in BuiltinSubagentFields) (BuiltinSubagentFields, bool) {
	out := in
	if in.Source != "builtin" {
		return out, false
	}
	def := defaultSubagentByFlag(in.Flag)
	if def == nil {
		return out, false
	}
	changed := false
	if in.SystemPrompt != def.SystemPrompt && slices.Contains(LegacyBuiltinSystemPrompts[in.Flag], in.SystemPrompt) {
		out.SystemPrompt = def.SystemPrompt
		changed = true
	}
	if in.Description != def.Description && slices.Contains(LegacyBuiltinDescriptions[in.Flag], in.Description) {
		out.Description = def.Description
		changed = true
	}
	return out, changed
}

// isMigratableBuiltinRow reports whether a store row is a known builtin seed candidate.
func isMigratableBuiltinRow(row *gen.AgentSubagent) bool {
	if row == nil || row.Source != "builtin" {
		return false
	}
	return defaultSubagentByFlag(row.Flag) != nil
}

func defaultSubagentByFlag(flag string) *gen.AgentSubagent {
	for _, item := range defaultSubagents {
		if item.Flag == flag {
			return item
		}
	}
	return nil
}

// defaultSubagents are seeded once when no subagent definitions exist, so the
// task tool is usable out of the box.
var defaultSubagents = []*gen.AgentSubagent{
	{
		Flag:        "general",
		Name:        "general",
		Description: "General-purpose subagent for complex research and multi-step tasks; can read and modify project files",
		SystemPrompt: `## Role
You are a general-purpose subagent. Complete the delegated task end to end in isolation.

## Constraints
- Use only the tools available to you, including reading and modifying project files when needed.
- Make reasonable assumptions; do not ask follow-up questions.
- Prefer tools over guessing file contents, command output, or external facts.

## Output
Return a concise, self-contained summary of what you found or did, including key file paths.`,
		Tools:   []string{"list_dir", "glob_files", "grep_files", "read_file", "write_file", "apply_patch", "web_search", "web_fetch", "run_terminal", "run_code"},
		Source:  "builtin",
		Enabled: true,
	},
	{
		Flag:        "explore",
		Name:        "explore",
		Description: "Fast read-only subagent for codebase navigation, locating implementations, and understanding complex logic",
		SystemPrompt: `## Role
You are an explore subagent operating in read-only mode. Navigate the codebase, locate implementations, and explain complex logic.

## Constraints
- Use only read-oriented tools (list_dir, glob_files, grep_files, read_file, web_search, web_fetch).
- Never modify files or attempt write-capable tools.
- Prefer tools over guessing file contents or architecture.

## Output
Return a concise, self-contained summary with file paths and relevant excerpts.`,
		Tools:   []string{"list_dir", "glob_files", "grep_files", "read_file", "web_search", "web_fetch"},
		Source:  "builtin",
		Enabled: true,
	},
	{
		Flag:        "scout",
		Name:        "scout",
		Description: "Research subagent for external docs, APIs, GitHub, and dependency investigation beyond the training cutoff",
		SystemPrompt: `## Role
You are a scout subagent focused on external information retrieval beyond the training cutoff.

## Constraints
- Prefer web_search and web_fetch for official docs, GitHub, and current API usage.
- Use run_terminal to clone or fetch external dependencies when needed, and read_file to inspect retrieved content.
- Cross-reference sources; do not invent citations or API details.

## Output
Return a concise, self-contained summary with links and actionable findings.`,
		Tools:   []string{"web_search", "web_fetch", "run_terminal", "read_file"},
		Source:  "builtin",
		Enabled: true,
	},
}

// SeedDefaultSubagents inserts built-in subagent definitions when none exist yet,
// and migrates legacy builtin prompts when they still match known shipped defaults.
func SeedDefaultSubagents(ctx context.Context) error {
	if store.Database == nil {
		return nil
	}
	existing, err := store.Database.ListAgentSubagents(ctx, false)
	if err != nil {
		return fmt.Errorf("list agent subagents: %w", err)
	}
	if len(existing) == 0 {
		now := time.Now().UTC()
		for _, item := range defaultSubagents {
			row := *item
			row.CreatedAt = now
			row.UpdatedAt = now
			if err := store.Database.CreateAgentSubagent(ctx, &row); err != nil {
				flog.Warn("[chat-agent] seed subagent %s: %v", row.Flag, err)
				continue
			}
			flog.Info("[chat-agent] seeded default subagent %s", row.Flag)
		}
		InvalidatePromptCache()
		return nil
	}
	return syncBuiltinSubagentPrompts(ctx, existing)
}

func syncBuiltinSubagentPrompts(ctx context.Context, existing []*gen.AgentSubagent) error {
	updated := false
	for _, row := range existing {
		if !isMigratableBuiltinRow(row) {
			continue
		}
		migrated, changed := MigrateBuiltinSubagentFields(BuiltinSubagentFields{
			Source:       row.Source,
			Flag:         row.Flag,
			SystemPrompt: row.SystemPrompt,
			Description:  row.Description,
		})
		if !changed {
			continue
		}
		next := *row
		next.SystemPrompt = migrated.SystemPrompt
		next.Description = migrated.Description
		if err := store.Database.UpdateAgentSubagent(ctx, &next); err != nil {
			flog.Warn("[chat-agent] migrate builtin subagent %s: %v", row.Flag, err)
			continue
		}
		flog.Info("[chat-agent] migrated builtin subagent prompt %s", row.Flag)
		updated = true
	}
	if updated {
		InvalidatePromptCache()
	}
	return nil
}

// FormatSubagentsForPrompt renders the available subagents in XML for the system prompt.
func FormatSubagentsForPrompt(subagents []Subagent) string {
	visible := make([]Subagent, 0, len(subagents))
	for _, item := range subagents {
		if strings.TrimSpace(item.Description) != "" {
			visible = append(visible, item)
		}
	}
	if len(visible) == 0 {
		return ""
	}

	lines := []string{
		"\n\nThe following subagents handle specialized tasks in an isolated context.",
		"Use the task tool with subagent_type set to a subagent name to delegate a self-contained task.",
		"Delegate when the task matches a subagent description; the subagent returns only its final result.",
		"",
		"<available_subagents>",
	}
	for _, item := range visible {
		lines = append(lines,
			"  <subagent>",
			fmt.Sprintf("    <name>%s</name>", escapeXML(item.Name)),
			fmt.Sprintf("    <description>%s</description>", escapeXML(item.Description)),
			"  </subagent>",
		)
	}
	lines = append(lines, "</available_subagents>")
	return strings.Join(lines, "\n")
}

// SelectableSubagentTools returns tool names available for subagent allowlist configuration.
func SelectableSubagentTools() []string {
	names := coding.ActiveToolNames()
	return append(names, updateMemoryToolName)
}

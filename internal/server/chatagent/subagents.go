package chatagent

import (
	"context"
	"fmt"
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

// defaultSubagents are seeded once when no subagent definitions exist, so the
// task tool is usable out of the box.
var defaultSubagents = []*gen.AgentSubagent{
	{
		Flag:        "general",
		Name:        "general",
		Description: "General-purpose subagent for complex research and multi-step tasks; can read and modify project files",
		SystemPrompt: "You are a general-purpose subagent. Complete the delegated task end to end using the available " +
			"tools, including reading and modifying project files when needed. Work in isolation, make reasonable " +
			"assumptions, and return a concise, self-contained summary of what you found or did. Do not ask " +
			"follow-up questions.",
		Tools:   []string{"read_file", "write_file", "web_search", "run_terminal", "run_code"},
		Source:  "builtin",
		Enabled: true,
	},
	{
		Flag:        "explore",
		Name:        "explore",
		Description: "Fast read-only subagent for codebase navigation, locating implementations, and understanding complex logic",
		SystemPrompt: "You are an explore subagent operating in read-only mode. Use read_file and web_search to " +
			"navigate the codebase, locate where features are implemented, and explain how complex logic works. " +
			"Never modify files or access write-capable tools. Return a concise, self-contained summary with " +
			"file paths and relevant excerpts.",
		Tools:   []string{"read_file", "web_search"},
		Source:  "builtin",
		Enabled: true,
	},
	{
		Flag:        "scout",
		Name:        "scout",
		Description: "Research subagent for external docs, APIs, GitHub, and dependency investigation beyond the training cutoff",
		SystemPrompt: "You are a scout subagent focused on external information retrieval. Use web_search to find " +
			"official documentation, GitHub repositories, and current API usage. Use run_terminal to clone or " +
			"fetch external dependencies when needed, and read_file to inspect retrieved content. Cross-reference " +
			"sources, verify facts against the latest docs, and return a concise, self-contained summary with " +
			"links and actionable findings.",
		Tools:   []string{"web_search", "run_terminal", "read_file"},
		Source:  "builtin",
		Enabled: true,
	},
}

// SeedDefaultSubagents inserts built-in subagent definitions when none exist yet.
// It is a best-effort startup helper and never overwrites user-managed rows.
func SeedDefaultSubagents(ctx context.Context) error {
	if store.Database == nil {
		return nil
	}
	existing, err := store.Database.ListAgentSubagents(ctx, false)
	if err != nil {
		return fmt.Errorf("list agent subagents: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}
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
	return coding.ActiveToolNames()
}

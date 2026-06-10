package chatagent

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const skillLocationPrefix = "skill://"

// Skill is a prompt-visible agent skill loaded from storage.
type Skill struct {
	Name                   string
	Description            string
	Location               string
	BaseDir                string
	DisableModelInvocation bool
}

// SkillContent is the full skill body returned by read_skill.
type SkillContent struct {
	Name    string
	Content string
	BaseDir string
}

// LoadSkillsFromStore loads enabled skills from the database.
func LoadSkillsFromStore(ctx context.Context) ([]Skill, error) {
	if store.Database == nil {
		return nil, nil
	}
	rows, err := store.Database.ListAgentSkills(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("load agent skills: %w", err)
	}
	skills := make([]Skill, 0, len(rows))
	for _, row := range rows {
		skills = append(skills, skillFromRow(row))
	}
	flog.Debug("[chat-agent] loaded %d skills from store", len(skills))
	return skills, nil
}

// GetSkillContent loads one enabled skill body by name.
func GetSkillContent(ctx context.Context, name string) (SkillContent, error) {
	if store.Database == nil {
		return SkillContent{}, fmt.Errorf("skill store unavailable")
	}
	row, err := store.Database.GetAgentSkillByName(ctx, name)
	if err != nil {
		return SkillContent{}, err
	}
	if row.DisableModelInvocation {
		return SkillContent{}, types.ErrForbidden
	}
	return SkillContent{
		Name:    row.Name,
		Content: row.Content,
		BaseDir: row.BaseDir,
	}, nil
}

func skillFromRow(row *gen.AgentSkill) Skill {
	location := skillLocationPrefix + row.Name
	if row.BaseDir != "" {
		location = normalizePromptPath(row.BaseDir + "/SKILL.md")
	}
	return Skill{
		Name:                   row.Name,
		Description:            row.Description,
		Location:               location,
		BaseDir:                row.BaseDir,
		DisableModelInvocation: row.DisableModelInvocation,
	}
}

// FormatSkillsForPrompt renders skills in XML for the system prompt.
func FormatSkillsForPrompt(skills []Skill) string {
	visible := make([]Skill, 0, len(skills))
	for _, skill := range skills {
		if !skill.DisableModelInvocation && strings.TrimSpace(skill.Description) != "" {
			visible = append(visible, skill)
		}
	}
	if len(visible) == 0 {
		return ""
	}

	lines := []string{
		"\n\nThe following skills provide specialized instructions for specific tasks.",
		"Use read_skill to load a skill when the task matches its description.",
		"When a skill references relative paths, resolve them against the skill base directory.",
		"",
		"<available_skills>",
	}
	for _, skill := range visible {
		lines = append(lines,
			"  <skill>",
			fmt.Sprintf("    <name>%s</name>", escapeXML(skill.Name)),
			fmt.Sprintf("    <description>%s</description>", escapeXML(skill.Description)),
			fmt.Sprintf("    <location>%s</location>", escapeXML(skill.Location)),
			"  </skill>",
		)
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n")
}

func escapeXML(value string) string {
	return html.EscapeString(value)
}

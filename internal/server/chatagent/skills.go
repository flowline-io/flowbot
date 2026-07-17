package chatagent

import (
	"context"
	"errors"
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const skillLocationPrefix = "skill://"

// legacySkillNames maps deprecated skill names/flags to hub.CapabilityType IDs.
// Kept for this release so existing DB rows and allowlists still resolve.
var legacySkillNames = map[string]string{
	"homelab-bookmark": "karakeep",
	"homelab-kanban":   "kanboard",
	"homelab-reader":   "miniflux",
}

// CanonicalSkillName returns the Cap ID form of a skill name.
// Unknown names are returned unchanged.
func CanonicalSkillName(name string) string {
	name = strings.TrimSpace(name)
	if mapped, ok := legacySkillNames[name]; ok {
		return mapped
	}
	return name
}

// skillNameCandidates returns lookup order for a requested skill name
// (requested, canonical Cap ID, and reverse legacy aliases).
func skillNameCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	seen := map[string]struct{}{name: {}}
	out := []string{name}
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(CanonicalSkillName(name))
	for old, neu := range legacySkillNames {
		if neu == name || neu == CanonicalSkillName(name) {
			add(old)
		}
	}
	return out
}

// skillNamesMatch reports whether a stored skill name matches an allowlist entry,
// including legacy homelab-* aliases.
func skillNamesMatch(skillName, allowedName string) bool {
	return CanonicalSkillName(skillName) == CanonicalSkillName(allowedName)
}

// Skill is a prompt-visible agent skill loaded from storage.
type Skill struct {
	Name                   string
	Description            string
	Location               string
	BaseDir                string
	Files                  []string
	DisableModelInvocation bool
}

// SkillContent is the full skill body returned by read_skill.
type SkillContent struct {
	Name    string
	Content string
	BaseDir string
	Files   []string
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
		skill := skillFromRow(row)
		files, err := store.Database.ListAgentSkillFiles(ctx, row.Flag)
		if err != nil {
			return nil, fmt.Errorf("load agent skill files for %q: %w", row.Name, err)
		}
		skill.Files = skillFilePaths(files)
		skills = append(skills, skill)
	}
	flog.Debug("[chat-agent] loaded %d skills from store", len(skills))
	return skills, nil
}

// GetSkillContent loads one enabled skill body by name.
func GetSkillContent(ctx context.Context, name string) (SkillContent, error) {
	if store.Database == nil {
		return SkillContent{}, fmt.Errorf("skill store unavailable")
	}
	row, err := getAgentSkillByName(ctx, name)
	if err != nil {
		return SkillContent{}, err
	}
	if row.DisableModelInvocation {
		return SkillContent{}, types.ErrForbidden
	}
	files, err := store.Database.ListAgentSkillFiles(ctx, row.Flag)
	if err != nil {
		return SkillContent{}, fmt.Errorf("load agent skill files: %w", err)
	}
	return SkillContent{
		Name:    row.Name,
		Content: row.Content,
		BaseDir: row.BaseDir,
		Files:   skillFilePaths(files),
	}, nil
}

// GetSkillFile loads one enabled skill auxiliary file by skill name and relative path.
func GetSkillFile(ctx context.Context, name, filePath string) (SkillContent, error) {
	if store.Database == nil {
		return SkillContent{}, fmt.Errorf("skill store unavailable")
	}
	normalized, err := normalizeSkillFilePath(filePath)
	if err != nil {
		return SkillContent{}, err
	}
	row, err := getAgentSkillByName(ctx, name)
	if err != nil {
		return SkillContent{}, err
	}
	if row.DisableModelInvocation {
		return SkillContent{}, types.ErrForbidden
	}
	file, err := store.Database.GetAgentSkillFile(ctx, row.Flag, normalized)
	if err != nil {
		return SkillContent{}, err
	}
	return SkillContent{
		Name:    row.Name,
		Content: file.Content,
		BaseDir: row.BaseDir,
	}, nil
}

// getAgentSkillByName loads a skill by name, trying Cap ID and legacy aliases.
func getAgentSkillByName(ctx context.Context, name string) (*gen.AgentSkill, error) {
	var lastErr error
	for _, candidate := range skillNameCandidates(name) {
		row, err := store.Database.GetAgentSkillByName(ctx, candidate)
		if err == nil {
			return row, nil
		}
		lastErr = err
		if !errors.Is(err, types.ErrNotFound) {
			return nil, err
		}
	}
	if lastErr == nil {
		return nil, types.ErrNotFound
	}
	return nil, lastErr
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

func skillFilePaths(files []*gen.AgentSkillFile) []string {
	if len(files) == 0 {
		return nil
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func normalizeSkillFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "skill file path is required")
	}
	if filepath.IsAbs(path) {
		return "", types.Errorf(types.ErrInvalidArgument, "skill file path must be relative")
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", types.Errorf(types.ErrInvalidArgument, "skill file path must stay within the skill directory")
	}
	return cleaned, nil
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
		"Use read_skill with the path argument to load auxiliary files listed under each skill.",
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
		)
		if len(skill.Files) > 0 {
			lines = append(lines, "    <files>")
			for _, filePath := range skill.Files {
				lines = append(lines, fmt.Sprintf("      <file>%s</file>", escapeXML(filePath)))
			}
			lines = append(lines, "    </files>")
		}
		lines = append(lines, "  </skill>")
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n")
}

// FilterSkillsByNames returns enabled skills whose names appear in allowlist.
// An empty allowlist returns no skills.
func FilterSkillsByNames(skills []Skill, allowlist []string) []Skill {
	if len(allowlist) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}
	filtered := make([]Skill, 0, len(allowlist))
	for _, skill := range skills {
		for allowedName := range allowed {
			if skillNamesMatch(skill.Name, allowedName) {
				filtered = append(filtered, skill)
				break
			}
		}
	}
	return filtered
}

func escapeXML(value string) string {
	return html.EscapeString(value)
}

func formatSkillContentText(content SkillContent) string {
	var b strings.Builder
	if content.BaseDir != "" {
		_, _ = fmt.Fprintf(&b, "Skill base directory: %s\n\n", content.BaseDir)
	}
	_, _ = b.WriteString(content.Content)
	if len(content.Files) > 0 {
		_, _ = b.WriteString("\n\nAvailable skill files (use read_skill with path):\n")
		for _, filePath := range content.Files {
			_, _ = fmt.Fprintf(&b, "- %s\n", filePath)
		}
	}
	return b.String()
}

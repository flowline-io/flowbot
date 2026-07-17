package chatagent

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/docs/skills"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	skillSourceBundled = "bundled"
)

// skillFrontmatter is the YAML header of a SKILL.md file.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ImportBundledSkills upserts embedded docs/skills into agent_skills / agent_skill_files.
// Safe to call on every server start; preserves enabled / disable_model_invocation on update.
func ImportBundledSkills(ctx context.Context) error {
	if store.Database == nil {
		return fmt.Errorf("skill store unavailable")
	}
	n, err := ImportSkillsFromFS(ctx, skills.FS, ".")
	if err != nil {
		return err
	}
	InvalidatePromptCache()
	flog.Info("[chat-agent] imported %d bundled skill(s) into agent_skills", n)
	return nil
}

// ImportSkillsFromFS walks root in fsys for */SKILL.md and upserts each skill.
func ImportSkillsFromFS(ctx context.Context, fsys fs.FS, root string) (int, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return 0, fmt.Errorf("read skill root %s: %w", root, err)
	}
	synced := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := path.Join(root, entry.Name())
		skillPath := path.Join(skillDir, "SKILL.md")
		if _, err := fs.Stat(fsys, skillPath); err != nil {
			continue
		}
		if err := upsertSkillFromFS(ctx, fsys, skillDir); err != nil {
			return synced, fmt.Errorf("import skill %s: %w", entry.Name(), err)
		}
		synced++
	}
	return synced, nil
}

func upsertSkillFromFS(ctx context.Context, fsys fs.FS, skillDir string) error {
	raw, err := fs.ReadFile(fsys, path.Join(skillDir, "SKILL.md"))
	if err != nil {
		return err
	}
	fm, body, err := parseSkillMarkdown(string(raw))
	if err != nil {
		return err
	}
	name := strings.TrimSpace(fm.Name)
	desc := strings.TrimSpace(fm.Description)
	body = strings.TrimSpace(body)
	switch {
	case name == "":
		return fmt.Errorf("SKILL.md missing name")
	case desc == "":
		return fmt.Errorf("SKILL.md missing description")
	case body == "":
		return fmt.Errorf("SKILL.md body is empty")
	}

	now := time.Now().UTC()
	existing, err := store.Database.GetAgentSkillByFlag(ctx, name)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}
	if errors.Is(err, types.ErrNotFound) {
		err = store.Database.CreateAgentSkill(ctx, &gen.AgentSkill{
			Flag:                   name,
			Name:                   name,
			Description:            desc,
			Content:                body,
			BaseDir:                skillDir,
			Source:                 skillSourceBundled,
			Enabled:                true,
			DisableModelInvocation: false,
			CreatedAt:              now,
			UpdatedAt:              now,
		})
		if err != nil {
			return fmt.Errorf("create skill %s: %w", name, err)
		}
	} else {
		existing.Name = name
		existing.Description = desc
		existing.Content = body
		existing.BaseDir = skillDir
		existing.Source = skillSourceBundled
		existing.UpdatedAt = now
		if err := store.Database.UpdateAgentSkill(ctx, existing); err != nil {
			return fmt.Errorf("update skill %s: %w", name, err)
		}
	}

	return syncReferenceFilesFromFS(ctx, fsys, name, path.Join(skillDir, "references"), now)
}

func syncReferenceFilesFromFS(ctx context.Context, fsys fs.FS, skillFlag, refsDir string, now time.Time) error {
	entries, err := fs.ReadDir(fsys, refsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		relPath := path.Join("references", entry.Name())
		content, err := fs.ReadFile(fsys, path.Join(refsDir, entry.Name()))
		if err != nil {
			return err
		}
		if err := upsertSkillFile(ctx, skillFlag, relPath, string(content), now); err != nil {
			return err
		}
	}
	return nil
}

func upsertSkillFile(ctx context.Context, skillFlag, filePath, content string, now time.Time) error {
	existing, err := store.Database.GetAgentSkillFile(ctx, skillFlag, filePath)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}
	if errors.Is(err, types.ErrNotFound) {
		return store.Database.CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
			SkillFlag: skillFlag,
			Path:      filePath,
			Content:   content,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	existing.Content = content
	existing.UpdatedAt = now
	return store.Database.UpdateAgentSkillFile(ctx, existing)
}

// parseSkillMarkdown splits SKILL.md into YAML frontmatter and markdown body.
func parseSkillMarkdown(raw string) (skillFrontmatter, string, error) {
	const delim = "---"
	trimmed := strings.TrimLeftFunc(raw, unicode.IsSpace)
	if !strings.HasPrefix(trimmed, delim) {
		return skillFrontmatter{}, "", fmt.Errorf("missing YAML frontmatter")
	}
	rest := strings.TrimPrefix(trimmed, delim)
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n"+delim)
	if end < 0 {
		return skillFrontmatter{}, "", fmt.Errorf("unterminated YAML frontmatter")
	}
	yamlBlock := rest[:end]
	body := strings.TrimLeft(rest[end+len("\n"+delim):], "\r\n")

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return skillFrontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}

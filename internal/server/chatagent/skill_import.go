package chatagent

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing/fstest"
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
	skillSourceBundled  = "bundled"
	skillSourceImported = "imported"

	maxSkillZipBytes        = 5 << 20 // 5 MiB compressed
	maxSkillZipFiles        = 256
	maxSkillZipUncompressed = 20 << 20 // 20 MiB total uncompressed
	maxSkillZipFileBytes    = 1 << 20  // 1 MiB per file
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

// ImportSkillsFromZip unpacks a skill zip archive and upserts each SKILL.md found.
// Source is recorded as "imported". Returns the number of skills synced.
func ImportSkillsFromZip(ctx context.Context, data []byte) (int, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty zip archive")
	}
	if len(data) > maxSkillZipBytes {
		return 0, fmt.Errorf("zip archive exceeds maximum size (%d bytes)", maxSkillZipBytes)
	}
	fsys, err := mapFSFromZip(data)
	if err != nil {
		return 0, err
	}
	n, err := upsertSkillsFromFS(ctx, fsys, ".", skillSourceImported)
	if err != nil {
		return n, err
	}
	if n > 0 {
		InvalidatePromptCache()
	}
	return n, nil
}

// ImportSkillsFromFS walks root in fsys for SKILL.md (including root) and upserts each skill.
func ImportSkillsFromFS(ctx context.Context, fsys fs.FS, root string) (int, error) {
	return upsertSkillsFromFS(ctx, fsys, root, skillSourceBundled)
}

func upsertSkillsFromFS(ctx context.Context, fsys fs.FS, root, source string) (int, error) {
	dirs, err := findSkillDirs(fsys, root)
	if err != nil {
		return 0, err
	}
	if len(dirs) == 0 {
		return 0, nil
	}
	if store.Database == nil {
		return 0, fmt.Errorf("skill store unavailable")
	}
	synced := 0
	for _, skillDir := range dirs {
		if err := upsertSkillFromFS(ctx, fsys, skillDir, source); err != nil {
			label := skillDir
			if label == "." || label == "" {
				label = "root"
			}
			return synced, fmt.Errorf("import skill %s: %w", label, err)
		}
		synced++
	}
	return synced, nil
}

// findSkillDirs returns directories under root that contain a SKILL.md file.
func findSkillDirs(fsys fs.FS, root string) ([]string, error) {
	var dirs []string
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if path.Base(p) != "SKILL.md" {
			return nil
		}
		dirs = append(dirs, path.Dir(p))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk skill root %s: %w", root, err)
	}
	slices.Sort(dirs)
	return dirs, nil
}

// mapFSFromZip materializes zip members into an in-memory FS with zip-slip protection.
func mapFSFromZip(data []byte) (fstest.MapFS, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	if len(zr.File) > maxSkillZipFiles {
		return nil, fmt.Errorf("zip archive has too many files")
	}
	out := make(fstest.MapFS)
	var total int64
	for _, f := range zr.File {
		name, err := sanitizeZipPath(f.Name)
		if err != nil {
			return nil, err
		}
		if name == "" || strings.HasSuffix(f.Name, "/") || f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > maxSkillZipFileBytes {
			return nil, fmt.Errorf("zip member %s exceeds maximum size", name)
		}
		total += int64(f.UncompressedSize64)
		if total > maxSkillZipUncompressed {
			return nil, fmt.Errorf("zip uncompressed size exceeds maximum")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip member %s: %w", name, err)
		}
		content, err := io.ReadAll(io.LimitReader(rc, maxSkillZipFileBytes+1))
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read zip member %s: %w", name, err)
		}
		if len(content) > maxSkillZipFileBytes {
			return nil, fmt.Errorf("zip member %s exceeds maximum size", name)
		}
		out[name] = &fstest.MapFile{Data: content}
	}
	return out, nil
}

// sanitizeZipPath rejects absolute paths and zip-slip (..) members.
func sanitizeZipPath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "./")
	if name == "" || name == "." {
		return "", nil
	}
	if path.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("illegal path in zip: %s", name)
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("illegal path in zip: %s", name)
	}
	return clean, nil
}

func upsertSkillFromFS(ctx context.Context, fsys fs.FS, skillDir, source string) error {
	skillMD := "SKILL.md"
	if skillDir != "." && skillDir != "" {
		skillMD = path.Join(skillDir, "SKILL.md")
	}
	raw, err := fs.ReadFile(fsys, skillMD)
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
	if source == "" {
		source = skillSourceBundled
	}
	baseDir := skillDir
	if baseDir == "." {
		baseDir = name
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
			BaseDir:                baseDir,
			Source:                 source,
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
		existing.BaseDir = baseDir
		existing.Source = source
		existing.UpdatedAt = now
		if err := store.Database.UpdateAgentSkill(ctx, existing); err != nil {
			return fmt.Errorf("update skill %s: %w", name, err)
		}
	}

	return syncSkillAuxFilesFromFS(ctx, fsys, name, skillDir, now)
}

// skillAuxFile is a bundled skill auxiliary file relative to the skill directory.
type skillAuxFile struct {
	RelPath string
	Content string
}

// isBundledSkillAuxFile reports whether rel (skill-dir relative) should be imported.
func isBundledSkillAuxFile(rel string) bool {
	rel = path.Clean(strings.TrimPrefix(rel, "./"))
	if rel == "SKILL.md" || rel == "." {
		return false
	}
	switch path.Ext(rel) {
	case ".md", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// collectSkillAuxFiles lists importable auxiliary files under skillDir (references, examples, …).
func collectSkillAuxFiles(fsys fs.FS, skillDir string) ([]skillAuxFile, error) {
	var out []skillAuxFile
	err := fs.WalkDir(fsys, skillDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, skillDir+"/")
		if rel == p {
			rel = strings.TrimPrefix(p, skillDir)
			rel = strings.TrimPrefix(rel, "/")
		}
		rel = path.Clean(rel)
		if rel == "." || !isBundledSkillAuxFile(rel) {
			return nil
		}
		content, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		out = append(out, skillAuxFile{RelPath: rel, Content: string(content)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b skillAuxFile) int {
		return strings.Compare(a.RelPath, b.RelPath)
	})
	return out, nil
}

func syncSkillAuxFilesFromFS(ctx context.Context, fsys fs.FS, skillFlag, skillDir string, now time.Time) error {
	files, err := collectSkillAuxFiles(fsys, skillDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := upsertSkillFile(ctx, skillFlag, f.RelPath, f.Content, now); err != nil {
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
	before, after, ok := strings.Cut(rest, "\n"+delim)
	if !ok {
		return skillFrontmatter{}, "", fmt.Errorf("unterminated YAML frontmatter")
	}
	yamlBlock := before
	body := strings.TrimLeft(after, "\r\n")

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return skillFrontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}

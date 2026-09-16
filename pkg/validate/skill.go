package validate

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

// Agent Skills limits from https://agentskills.io/specification (skills-ref).
const (
	SkillNameMaxLen          = 64
	SkillDescMaxLen          = 1024
	SkillCompatibilityMaxLen = 500
)

// SkillFrontmatter is the YAML header of a SKILL.md file.
type SkillFrontmatter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Compatibility string `yaml:"compatibility"`
}

// ParseSkillMarkdown splits SKILL.md into YAML frontmatter and markdown body.
func ParseSkillMarkdown(raw string) (SkillFrontmatter, string, error) {
	const delim = "---"
	trimmed := strings.TrimLeftFunc(raw, unicode.IsSpace)
	if !strings.HasPrefix(trimmed, delim) {
		return SkillFrontmatter{}, "", errors.New("missing YAML frontmatter")
	}
	rest := strings.TrimPrefix(trimmed, delim)
	rest = strings.TrimLeft(rest, "\r\n")
	before, after, ok := strings.Cut(rest, "\n"+delim)
	if !ok {
		return SkillFrontmatter{}, "", errors.New("unterminated YAML frontmatter")
	}
	var fm SkillFrontmatter
	if err := yaml.Unmarshal([]byte(before), &fm); err != nil {
		return SkillFrontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, strings.TrimLeft(after, "\r\n"), nil
}

// SkillName checks a skill name against the Agent Skills naming rules.
//
// Rules: 1–64 characters; lowercase letters, digits, and hyphens only; must not
// start or end with a hyphen; must not contain consecutive hyphens.
func SkillName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("skill name must be a non-empty string")
	}
	if utf8.RuneCountInString(name) > SkillNameMaxLen {
		return fmt.Errorf("skill name %q exceeds %d character limit (%d chars)", name, SkillNameMaxLen, utf8.RuneCountInString(name))
	}
	if name != strings.ToLower(name) {
		return fmt.Errorf("skill name %q must be lowercase", name)
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return errors.New("skill name cannot start or end with a hyphen")
	}
	if strings.Contains(name, "--") {
		return errors.New("skill name cannot contain consecutive hyphens")
	}
	for _, r := range name {
		if r == '-' {
			continue
		}
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r)) {
			return fmt.Errorf("skill name %q contains invalid characters (only a-z, 0-9, and hyphens are allowed)", name)
		}
	}
	return nil
}

// SkillDescription checks a skill description (required, ≤ SkillDescMaxLen runes).
func SkillDescription(description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("skill description must be a non-empty string")
	}
	if n := utf8.RuneCountInString(description); n > SkillDescMaxLen {
		return fmt.Errorf("skill description exceeds %d character limit (%d chars)", SkillDescMaxLen, n)
	}
	return nil
}

// SkillCompatibility checks an optional compatibility field (≤ SkillCompatibilityMaxLen runes).
func SkillCompatibility(compatibility string) error {
	if strings.TrimSpace(compatibility) == "" {
		return nil
	}
	if n := utf8.RuneCountInString(compatibility); n > SkillCompatibilityMaxLen {
		return fmt.Errorf("skill compatibility exceeds %d character limit (%d chars)", SkillCompatibilityMaxLen, n)
	}
	return nil
}

// SkillNameMatchesDir reports whether name equals the skill directory basename.
// dirName of "" or "." skips the check (root-level SKILL.md imports).
func SkillNameMatchesDir(name, dirName string) error {
	name = strings.TrimSpace(name)
	dirName = strings.TrimSpace(dirName)
	if dirName == "" || dirName == "." {
		return nil
	}
	if dirName != name {
		return fmt.Errorf("directory name %q must match skill name %q", dirName, name)
	}
	return nil
}

// SkillIdentity checks name, description, and optional directory match together.
func SkillIdentity(name, description, dirName string) error {
	if err := SkillName(name); err != nil {
		return err
	}
	if err := SkillDescription(description); err != nil {
		return err
	}
	return SkillNameMatchesDir(name, dirName)
}

// SkillDocument checks Agent Skills identity fields including optional compatibility.
func SkillDocument(name, description, compatibility, dirName string) error {
	if err := SkillIdentity(name, description, dirName); err != nil {
		return err
	}
	return SkillCompatibility(compatibility)
}

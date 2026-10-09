package pipeline

import (
	"cmp"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

//go:embed blueprints/*.yaml
var builtinBlueprintFS embed.FS

// BuiltinBlueprint holds one official catalog YAML and its parsed document.
type BuiltinBlueprint struct {
	Document BlueprintDocument
	YAML     string
	Hash     string
}

// LoadBuiltinBlueprints reads the embedded official catalog.
func LoadBuiltinBlueprints() ([]BuiltinBlueprint, error) {
	entries, err := fs.ReadDir(builtinBlueprintFS, "blueprints")
	if err != nil {
		return nil, fmt.Errorf("read builtin blueprints: %w", err)
	}
	out := make([]BuiltinBlueprint, 0, len(entries))
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".yaml") {
			continue
		}
		data, err := builtinBlueprintFS.ReadFile(path.Join("blueprints", ent.Name()))
		if err != nil {
			return nil, fmt.Errorf("read builtin blueprint %s: %w", ent.Name(), err)
		}
		doc, err := ParseBlueprintYAML(data)
		if err != nil {
			return nil, fmt.Errorf("parse builtin blueprint %s: %w", ent.Name(), err)
		}
		out = append(out, BuiltinBlueprint{
			Document: *doc,
			YAML:     string(data),
			Hash:     ContentHash(data),
		})
	}
	slices.SortFunc(out, func(a, b BuiltinBlueprint) int {
		return cmp.Compare(a.Document.ID, b.Document.ID)
	})
	return out, nil
}

// LookupBuiltinBlueprint returns the official template with the given id.
func LookupBuiltinBlueprint(id string) (*BuiltinBlueprint, error) {
	all, err := LoadBuiltinBlueprints()
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	for i := range all {
		if all[i].Document.ID == id {
			cp := all[i]
			return &cp, nil
		}
	}
	return nil, types.Errorf(types.ErrNotFound, "builtin blueprint %s not found", id)
}

// BuiltinBlueprintIDs returns the set of official blueprint ids.
func BuiltinBlueprintIDs() (map[string]struct{}, error) {
	all, err := LoadBuiltinBlueprints()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]struct{}, len(all))
	for _, b := range all {
		ids[b.Document.ID] = struct{}{}
	}
	return ids, nil
}

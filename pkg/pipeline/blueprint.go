package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// Blueprint document kind stored in YAML.
const KindPipelineBlueprint = "pipeline_blueprint"

// Origin sources for a linked pipeline instance.
const (
	BlueprintSourceBuiltin = "builtin"
	BlueprintSourceUpload  = "upload"
)

// Closed set of blueprint input types.
const (
	BlueprintInputString        = "string"
	BlueprintInputNumber        = "number"
	BlueprintInputBoolean       = "boolean"
	BlueprintInputNotifyChannel = "notify_channel"
)

// BlueprintIDPattern matches a stable blueprint identifier.
var BlueprintIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

var blueprintInputPlaceholder = regexp.MustCompile(`\{\{\s*inputs\.([A-Za-z][A-Za-z0-9_]*)\s*\}\}`)

// BlueprintRequires lists capabilities the template needs in order to instantiate.
type BlueprintRequires struct {
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// BlueprintInputDef declares one instance-configuration parameter.
type BlueprintInputDef struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// BlueprintDocument is the on-disk / import wrapper for a pipeline blueprint.
type BlueprintDocument struct {
	Kind        string              `json:"kind" yaml:"kind"`
	ID          string              `json:"id" yaml:"id"`
	Title       string              `json:"title" yaml:"title"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Requires    BlueprintRequires   `json:"requires" yaml:"requires,omitempty"`
	Inputs      []BlueprintInputDef `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Definition  map[string]any      `json:"definition" yaml:"definition"`
}

// BlueprintOrigin is persisted on a linked pipeline definition.
type BlueprintOrigin struct {
	Source string         `json:"source"`
	ID     string         `json:"id"`
	Hash   string         `json:"hash"`
	YAML   string         `json:"yaml"`
	Inputs map[string]any `json:"inputs"`
}

// BlueprintTemplateRecord is a user-library row.
type BlueprintTemplateRecord struct {
	BlueprintID string
	Title       string
	Description string
	YAML        string
	Hash        string
	CreatedBy   string
}

// BlueprintCatalogEntry is one card in the template library.
type BlueprintCatalogEntry struct {
	Source          string
	Document        BlueprintDocument
	YAML            string
	Hash            string
	Missing         []string
	InstanceCount   int
	UpdateAvailable bool
}

// ContentHash returns the SHA-256 hex digest of blueprint YAML bytes.
func ContentHash(yamlBytes []byte) string {
	sum := sha256.Sum256(yamlBytes)
	return hex.EncodeToString(sum[:])
}

// ValidateBlueprintID reports whether id is a valid blueprint identifier.
func ValidateBlueprintID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "blueprint id is required")
	}
	if !BlueprintIDPattern.MatchString(id) {
		return types.Errorf(types.ErrInvalidArgument, "blueprint id must be lowercase letters, digits, and underscores, starting with a letter")
	}
	return nil
}

// LooksLikeBlueprint reports whether YAML declares kind: pipeline_blueprint.
func LooksLikeBlueprint(yamlBytes []byte) bool {
	var probe struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(yamlBytes, &probe); err != nil {
		return false
	}
	return probe.Kind == KindPipelineBlueprint
}

// MissingCapabilities returns required capability types that are not registered on the hub.
func MissingCapabilities(requires []string) []string {
	var missing []string
	seen := map[string]bool{}
	for _, raw := range requires {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if _, ok := hub.Default.Get(hub.CapabilityType(id)); !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func validInputType(t string) bool {
	switch t {
	case BlueprintInputString, BlueprintInputNumber, BlueprintInputBoolean, BlueprintInputNotifyChannel:
		return true
	default:
		return false
	}
}

func validateBlueprintDocument(doc *BlueprintDocument) error {
	if doc == nil {
		return types.Errorf(types.ErrInvalidArgument, "blueprint document is nil")
	}
	if doc.Kind != KindPipelineBlueprint {
		return types.Errorf(types.ErrInvalidArgument, "kind must be %s", KindPipelineBlueprint)
	}
	if err := ValidateBlueprintID(doc.ID); err != nil {
		return err
	}
	if strings.TrimSpace(doc.Title) == "" {
		return types.Errorf(types.ErrInvalidArgument, "blueprint title is required")
	}
	if len(doc.Definition) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "blueprint definition is required")
	}
	seen := map[string]bool{}
	for _, in := range doc.Inputs {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return types.Errorf(types.ErrInvalidArgument, "input name is required")
		}
		if seen[name] {
			return types.Errorf(types.ErrInvalidArgument, "duplicate input %s", name)
		}
		seen[name] = true
		if !validInputType(in.Type) {
			return types.Errorf(types.ErrInvalidArgument, "unsupported input type %q", in.Type)
		}
	}
	return nil
}

func originLinked(src, id string) bool {
	return strings.TrimSpace(src) != "" && strings.TrimSpace(id) != ""
}

// OriginFromDefinition returns origin metadata when the pipeline is still linked.
func OriginFromDefinition(def *model.PipelineDefinition) (BlueprintOrigin, bool) {
	if def == nil || !originLinked(def.BlueprintSource, def.BlueprintID) {
		return BlueprintOrigin{}, false
	}
	return BlueprintOrigin{
		Source: def.BlueprintSource,
		ID:     def.BlueprintID,
		Hash:   def.BlueprintHash,
		YAML:   def.BlueprintYAML,
		Inputs: def.BlueprintInputs,
	}, true
}

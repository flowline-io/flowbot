package pipeline

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// IsEnabledInYAML reports whether a pipeline definition YAML is active.
// Missing or unparseable enabled fields default to true.
func IsEnabledInYAML(yamlStr string) bool {
	if yamlStr == "" {
		return true
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
		return true
	}
	v, ok := raw["enabled"]
	if !ok {
		return true
	}
	enabled, ok := v.(bool)
	if !ok {
		return true
	}
	return enabled
}

// SetEnabledInYAML returns YAML with the top-level enabled field updated.
// Cron triggers are synced so pause also stops scheduled runs after engine reload.
func SetEnabledInYAML(yamlStr string, enabled bool) (string, error) {
	if yamlStr == "" {
		return "", fmt.Errorf("set enabled in yaml: empty input")
	}
	def, err := ParseEditorYAML(yamlStr)
	if err != nil {
		return "", fmt.Errorf("set enabled in yaml: %w", err)
	}
	def.Enabled = enabled
	syncCronTriggersEnabled(def, enabled)
	out, err := yaml.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("set enabled in yaml: marshal: %w", err)
	}
	return string(out), nil
}

func syncCronTriggersEnabled(def *EditorDefinition, enabled bool) {
	for i := range def.Triggers {
		if def.Triggers[i].Type == "cron" {
			def.Triggers[i].Enabled = enabled
		}
	}
}

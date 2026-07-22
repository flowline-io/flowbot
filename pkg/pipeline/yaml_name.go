package pipeline

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// SetNameInYAML returns YAML with the top-level name field updated.
func SetNameInYAML(yamlStr, name string) (string, error) {
	if yamlStr == "" {
		return "", fmt.Errorf("set name in yaml: empty input")
	}
	if err := ValidateName(name); err != nil {
		return "", fmt.Errorf("set name in yaml: %w", err)
	}
	def, err := ParseEditorYAML(yamlStr)
	if err != nil {
		return "", fmt.Errorf("set name in yaml: %w", err)
	}
	def.Name = name
	out, err := yaml.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("set name in yaml: marshal: %w", err)
	}
	return string(out), nil
}

package pipeline

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ParseBlueprintYAML parses and validates a pipeline blueprint document.
func ParseBlueprintYAML(yamlBytes []byte) (*BlueprintDocument, error) {
	if len(strings.TrimSpace(string(yamlBytes))) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "blueprint YAML is empty")
	}
	var doc BlueprintDocument
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "parse blueprint YAML", err)
	}
	if err := validateBlueprintDocument(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// MarshalBlueprintYAML serializes a blueprint document.
func MarshalBlueprintYAML(doc *BlueprintDocument) ([]byte, error) {
	if err := validateBlueprintDocument(doc); err != nil {
		return nil, err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal blueprint YAML: %w", err)
	}
	return out, nil
}

package schema

import (
	"fmt"
	"regexp"
)

// PipelineNamePattern matches valid pipeline names: Unicode letters and digits,
// with optional underscores and hyphens. Must start with a letter or digit.
var PipelineNamePattern = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}_-]*$`)

// ValidatePipelineName reports whether name is a valid pipeline identifier.
func ValidatePipelineName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !PipelineNamePattern.MatchString(name) {
		return fmt.Errorf("name must start with a letter or digit and contain only letters, digits, underscores, or hyphens")
	}
	return nil
}

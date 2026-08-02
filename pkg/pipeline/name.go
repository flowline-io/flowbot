package pipeline

import "github.com/flowline-io/flowbot/pkg/types"

// NamePattern matches valid pipeline names.
var NamePattern = types.PipelineNamePattern

// ValidateName reports whether name is a valid pipeline identifier.
func ValidateName(name string) error {
	return types.ValidatePipelineName(name)
}

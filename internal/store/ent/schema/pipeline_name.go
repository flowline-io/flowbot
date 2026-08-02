package schema

import "github.com/flowline-io/flowbot/pkg/types"

// PipelineNamePattern matches valid pipeline names.
// Owned by pkg/types; alias keeps schema.* call sites.
var PipelineNamePattern = types.PipelineNamePattern

// ValidatePipelineName reports whether name is a valid pipeline identifier.
func ValidatePipelineName(name string) error {
	return types.ValidatePipelineName(name)
}

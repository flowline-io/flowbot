package pipeline

import "github.com/flowline-io/flowbot/internal/store/ent/schema"

// NamePattern matches valid pipeline names.
var NamePattern = schema.PipelineNamePattern

// ValidateName reports whether name is a valid pipeline identifier.
func ValidateName(name string) error {
	return schema.ValidatePipelineName(name)
}

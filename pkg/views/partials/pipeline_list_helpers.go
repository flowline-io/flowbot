package partials

import (
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)

// PipelineListEntry augments a pipeline definition with runtime enabled state.
type PipelineListEntry struct {
	Definition *gen.PipelineDefinition
	Enabled    bool
}

// BuildPipelineListEntries derives list rows from stored pipeline definitions.
func BuildPipelineListEntries(defs []*gen.PipelineDefinition) []PipelineListEntry {
	entries := make([]PipelineListEntry, 0, len(defs))
	for _, def := range defs {
		yaml := def.YamlDraft
		if def.Status == pipelinedefinition.StatusPublished && def.YamlPublished != nil && *def.YamlPublished != "" {
			yaml = *def.YamlPublished
		}
		entries = append(entries, PipelineListEntry{
			Definition: def,
			Enabled:    pipeline.IsEnabledInYAML(yaml),
		})
	}
	return entries
}

// PipelineIsPublished reports whether a pipeline has a published runtime definition.
func PipelineIsPublished(def *gen.PipelineDefinition) bool {
	return def != nil &&
		def.Status == pipelinedefinition.StatusPublished &&
		def.YamlPublished != nil &&
		*def.YamlPublished != ""
}

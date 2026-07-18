package partials

import (
	"net/url"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)

// PipelineWebPath returns the encoded web UI path for a pipeline name.
func PipelineWebPath(name string) string {
	return "/service/web/pipelines/" + url.PathEscape(name)
}

// PipelineListEntry augments a pipeline definition with runtime enabled state and last run time.
type PipelineListEntry struct {
	Definition *gen.PipelineDefinition
	Enabled    bool
	LastRunAt  *time.Time
}

// BuildPipelineListEntries derives list rows from stored pipeline definitions.
// lastRunAt maps parent pipeline name to the latest run started_at; missing keys mean never run.
func BuildPipelineListEntries(defs []*gen.PipelineDefinition, lastRunAt map[string]time.Time) []PipelineListEntry {
	entries := make([]PipelineListEntry, 0, len(defs))
	for _, def := range defs {
		yaml := def.YamlDraft
		if def.Status == pipelinedefinition.StatusPublished && def.YamlPublished != nil && *def.YamlPublished != "" {
			yaml = *def.YamlPublished
		}
		var last *time.Time
		if t, ok := lastRunAt[def.Name]; ok {
			lastCopy := t
			last = &lastCopy
		}
		entries = append(entries, PipelineListEntry{
			Definition: def,
			Enabled:    pipeline.IsEnabledInYAML(yaml),
			LastRunAt:  last,
		})
	}
	return entries
}

// PipelineLastRunOrDash formats an optional last-run timestamp for table cells.
func PipelineLastRunOrDash(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "—"
	}
	return value.Format("2006-01-02 15:04")
}

// PipelineIsPublished reports whether a pipeline has a published runtime definition.
func PipelineIsPublished(def *gen.PipelineDefinition) bool {
	return def != nil &&
		def.Status == pipelinedefinition.StatusPublished &&
		def.YamlPublished != nil &&
		*def.YamlPublished != ""
}

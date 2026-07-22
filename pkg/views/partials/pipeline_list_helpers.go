package partials

import (
	"net/url"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PipelineWebPath returns the encoded web UI path for a pipeline name.
func PipelineWebPath(name string) string {
	return "/service/web/pipelines/" + url.PathEscape(name)
}

// PipelineTriggerSummary describes one configured trigger for the pipelines list.
type PipelineTriggerSummary struct {
	// Type is the trigger kind: event, cron, or webhook.
	Type string
	// Label is the tooltip text (event name, cron expression, or webhook path).
	Label string
	// Enabled reports whether the trigger is active in the definition.
	Enabled bool
	// Letter is the single-character badge shown in the list (E / C / W).
	Letter string
}

// PipelineListEntry augments a pipeline definition with runtime enabled state and last run time.
type PipelineListEntry struct {
	Definition *gen.PipelineDefinition
	Enabled    bool
	LastRunAt  *time.Time
	// Triggers lists configured triggers from the displayed YAML (published when published).
	Triggers []PipelineTriggerSummary
	// StepCount is the number of steps in the displayed YAML.
	StepCount int
	// Stats holds recent completed-run latency aggregates when available.
	Stats *types.RunLatencyStats
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
		stepCount, triggers := PipelineListSummaryFromYAML(yaml)
		entries = append(entries, PipelineListEntry{
			Definition: def,
			Enabled:    pipeline.IsEnabledInYAML(yaml),
			LastRunAt:  last,
			Triggers:   triggers,
			StepCount:  stepCount,
		})
	}
	return entries
}

// PipelineListSummaryFromYAML extracts step count and trigger summaries from editor YAML.
// Unparseable or empty YAML yields zero steps and a nil trigger slice.
func PipelineListSummaryFromYAML(yamlStr string) (int, []PipelineTriggerSummary) {
	if yamlStr == "" {
		return 0, nil
	}
	def, err := pipeline.ParseEditorYAML(yamlStr)
	if err != nil || def == nil {
		return 0, nil
	}
	triggers := make([]PipelineTriggerSummary, 0, len(def.Triggers))
	for _, t := range def.Triggers {
		triggers = append(triggers, PipelineTriggerSummary{
			Type:    t.Type,
			Label:   pipelineTriggerLabel(t),
			Enabled: t.Enabled,
			Letter:  PipelineTriggerLetter(t.Type),
		})
	}
	return len(def.Steps), triggers
}

// PipelineTriggerLetter returns the list badge letter for a trigger type.
func PipelineTriggerLetter(typ string) string {
	switch typ {
	case "event":
		return "E"
	case "cron":
		return "C"
	case "webhook":
		return "W"
	case "manual":
		return "M"
	default:
		return "?"
	}
}

func pipelineTriggerLabel(t pipeline.TriggerEntry) string {
	switch t.Type {
	case "event":
		if t.Event == "" {
			return "Event"
		}
		return "Event: " + t.Event
	case "cron":
		if t.Cron == "" {
			return "Cron"
		}
		return "Cron: " + t.Cron
	case "webhook":
		path := ""
		if t.Webhook != nil {
			path = t.Webhook.Path
		}
		if path == "" {
			return "Webhook"
		}
		return "Webhook: " + path
	default:
		if t.Type == "" {
			return "Trigger"
		}
		return t.Type
	}
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

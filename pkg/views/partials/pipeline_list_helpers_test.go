package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
)

func TestBuildPipelineListEntries(t *testing.T) {
	publishedYAML := "name: pub\nenabled: false\ntriggers:\n  - type: cron\n    enabled: true\n    cron: '@daily'\nsteps:\n  - name: a\n"
	draftYAML := "name: draft\nenabled: true\ntriggers:\n  - type: event\n    enabled: true\n    event: bookmark.created\nsteps:\n  - name: a\n  - name: b\n"
	published := pipelinedefinition.Status("published")
	draft := pipelinedefinition.Status("draft")
	lastRun := time.Date(2026, 7, 18, 14, 22, 0, 0, time.UTC)

	tests := []struct {
		name          string
		defs          []*gen.PipelineDefinition
		lastRunAt     map[string]time.Time
		wantCount     int
		wantFirst     bool
		wantLastRun   *time.Time
		wantStepCount int
		wantTriggers  []string
	}{
		{
			name:      "empty list",
			defs:      nil,
			wantCount: 0,
		},
		{
			name: "draft uses draft yaml and has no last run",
			defs: []*gen.PipelineDefinition{{
				Name:      "draft-only",
				Status:    draft,
				YamlDraft: draftYAML,
			}},
			wantCount:     1,
			wantFirst:     true,
			wantLastRun:   nil,
			wantStepCount: 2,
			wantTriggers:  []string{"event"},
		},
		{
			name: "published uses published yaml and attaches last run",
			defs: []*gen.PipelineDefinition{{
				Name:          "paused",
				Status:        published,
				YamlDraft:     draftYAML,
				YamlPublished: &publishedYAML,
			}},
			lastRunAt:     map[string]time.Time{"paused": lastRun},
			wantCount:     1,
			wantFirst:     false,
			wantLastRun:   &lastRun,
			wantStepCount: 1,
			wantTriggers:  []string{"cron"},
		},
		{
			name: "multiple entries preserve order",
			defs: []*gen.PipelineDefinition{
				{Name: "a", Status: draft, YamlDraft: draftYAML},
				{Name: "b", Status: published, YamlDraft: draftYAML, YamlPublished: &publishedYAML},
			},
			wantCount:     2,
			wantFirst:     true,
			wantStepCount: 2,
			wantTriggers:  []string{"event"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPipelineListEntries(tt.defs, tt.lastRunAt)
			require.Len(t, got, tt.wantCount)
			if tt.wantCount == 0 {
				return
			}
			assert.Equal(t, tt.wantFirst, got[0].Enabled)
			assert.Equal(t, tt.wantStepCount, got[0].StepCount)
			gotTypes := make([]string, len(got[0].Triggers))
			for i, tr := range got[0].Triggers {
				gotTypes[i] = tr.Type
			}
			assert.Equal(t, tt.wantTriggers, gotTypes)
			if tt.wantLastRun == nil {
				assert.Nil(t, got[0].LastRunAt)
			} else {
				require.NotNil(t, got[0].LastRunAt)
				assert.True(t, got[0].LastRunAt.Equal(*tt.wantLastRun))
			}
		})
	}
}

func TestPipelineListSummaryFromYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		yaml          string
		wantStepCount int
		wantTriggers  []PipelineTriggerSummary
	}{
		{
			name:          "empty yaml yields zero steps and no triggers",
			yaml:          "",
			wantStepCount: 0,
			wantTriggers:  nil,
		},
		{
			name: "event cron and webhook triggers with labels",
			yaml: `name: multi
triggers:
  - type: event
    enabled: true
    event: bookmark.created
  - type: cron
    enabled: false
    cron: "0 */6 * * *"
  - type: webhook
    enabled: true
    webhook:
      path: /hooks/gh
steps:
  - name: one
  - name: two
  - name: three
`,
			wantStepCount: 3,
			wantTriggers: []PipelineTriggerSummary{
				{Type: "event", Label: "Event: bookmark.created", Enabled: true, Letter: "E"},
				{Type: "cron", Label: "Cron: 0 */6 * * *", Enabled: false, Letter: "C"},
				{Type: "webhook", Label: "Webhook: /hooks/gh", Enabled: true, Letter: "W"},
			},
		},
		{
			name:          "invalid yaml yields empty summary",
			yaml:          ": not valid",
			wantStepCount: 0,
			wantTriggers:  nil,
		},
		{
			name: "unknown trigger type still listed with letter",
			yaml: `name: odd
triggers:
  - type: custom
    enabled: true
steps: []
`,
			wantStepCount: 0,
			wantTriggers: []PipelineTriggerSummary{
				{Type: "custom", Label: "custom", Enabled: true, Letter: "?"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			steps, triggers := PipelineListSummaryFromYAML(tt.yaml)
			assert.Equal(t, tt.wantStepCount, steps)
			assert.Equal(t, tt.wantTriggers, triggers)
		})
	}
}

func TestPipelineTriggerLetter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{name: "event", typ: "event", want: "E"},
		{name: "cron", typ: "cron", want: "C"},
		{name: "webhook", typ: "webhook", want: "W"},
		{name: "manual", typ: "manual", want: "M"},
		{name: "unknown", typ: "other", want: "?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PipelineTriggerLetter(tt.typ))
		})
	}
}

func TestPipelineLastRunOrDash(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 7, 18, 14, 22, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value *time.Time
		want  string
	}{
		{name: "nil", value: nil, want: "—"},
		{name: "zero", value: &time.Time{}, want: "—"},
		{name: "formatted", value: &ts, want: "2026-07-18 14:22"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PipelineLastRunOrDash(tt.value))
		})
	}
}

func TestPipelineWebPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "ascii name unchanged",
			in:   "my-pipeline",
			want: "/service/web/pipelines/my-pipeline",
		},
		{
			name: "chinese name encoded",
			in:   "数据同步",
			want: "/service/web/pipelines/%E6%95%B0%E6%8D%AE%E5%90%8C%E6%AD%A5",
		},
		{
			name: "mixed name encoded",
			in:   "同步-bookmarks",
			want: "/service/web/pipelines/%E5%90%8C%E6%AD%A5-bookmarks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PipelineWebPath(tt.in))
		})
	}
}

func TestPipelineIsPublished(t *testing.T) {
	yaml := "name: test"
	tests := []struct {
		name string
		def  *gen.PipelineDefinition
		want bool
	}{
		{name: "nil definition", def: nil, want: false},
		{name: "draft only", def: &gen.PipelineDefinition{Status: pipelinedefinition.Status("draft")}, want: false},
		{
			name: "published without yaml",
			def:  &gen.PipelineDefinition{Status: pipelinedefinition.Status("published")},
			want: false,
		},
		{
			name: "published with yaml",
			def:  &gen.PipelineDefinition{Status: pipelinedefinition.Status("published"), YamlPublished: &yaml},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PipelineIsPublished(tt.def))
		})
	}
}

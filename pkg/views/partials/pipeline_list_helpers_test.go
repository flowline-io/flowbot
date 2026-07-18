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
	publishedYAML := "name: pub\nenabled: false\nsteps: []"
	draftYAML := "name: draft\nenabled: true\nsteps: []"
	published := pipelinedefinition.Status("published")
	draft := pipelinedefinition.Status("draft")
	lastRun := time.Date(2026, 7, 18, 14, 22, 0, 0, time.UTC)

	tests := []struct {
		name        string
		defs        []*gen.PipelineDefinition
		lastRunAt   map[string]time.Time
		wantCount   int
		wantFirst   bool
		wantLastRun *time.Time
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
			wantCount:   1,
			wantFirst:   true,
			wantLastRun: nil,
		},
		{
			name: "published uses published yaml and attaches last run",
			defs: []*gen.PipelineDefinition{{
				Name:          "paused",
				Status:        published,
				YamlDraft:     draftYAML,
				YamlPublished: &publishedYAML,
			}},
			lastRunAt:   map[string]time.Time{"paused": lastRun},
			wantCount:   1,
			wantFirst:   false,
			wantLastRun: &lastRun,
		},
		{
			name: "multiple entries preserve order",
			defs: []*gen.PipelineDefinition{
				{Name: "a", Status: draft, YamlDraft: draftYAML},
				{Name: "b", Status: published, YamlDraft: draftYAML, YamlPublished: &publishedYAML},
			},
			wantCount: 2,
			wantFirst: true,
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
			if tt.wantLastRun == nil {
				assert.Nil(t, got[0].LastRunAt)
			} else {
				require.NotNil(t, got[0].LastRunAt)
				assert.True(t, got[0].LastRunAt.Equal(*tt.wantLastRun))
			}
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

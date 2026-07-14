package partials

import (
	"testing"

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

	tests := []struct {
		name      string
		defs      []*gen.PipelineDefinition
		wantCount int
		wantFirst bool
	}{
		{
			name:      "empty list",
			defs:      nil,
			wantCount: 0,
		},
		{
			name: "draft uses draft yaml",
			defs: []*gen.PipelineDefinition{{
				Name:      "draft-only",
				Status:    draft,
				YamlDraft: draftYAML,
			}},
			wantCount: 1,
			wantFirst: true,
		},
		{
			name: "published uses published yaml",
			defs: []*gen.PipelineDefinition{{
				Name:          "paused",
				Status:        published,
				YamlDraft:     draftYAML,
				YamlPublished: &publishedYAML,
			}},
			wantCount: 1,
			wantFirst: false,
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
			got := BuildPipelineListEntries(tt.defs)
			require.Len(t, got, tt.wantCount)
			if tt.wantCount == 0 {
				return
			}
			assert.Equal(t, tt.wantFirst, got[0].Enabled)
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

package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestWorkflowPollingPreservesScroll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		render func() (string, error)
		want   []string
	}{
		{
			name: "workflow list panel polling uses show none and preserve scroll",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := pages.WorkflowListPage(nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				`id="workflow-list-panel"`,
				`hx-swap="innerHTML show:none"`,
				`data-preserve-scroll`,
			},
		},
		{
			name: "workflow runs panel polling uses show none and preserve scroll",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := pages.WorkflowRunsPage("demo", nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				`id="workflow-runs-panel"`,
				`hx-swap="innerHTML show:none"`,
				`data-preserve-scroll`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html, err := tt.render()
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
		})
	}
}

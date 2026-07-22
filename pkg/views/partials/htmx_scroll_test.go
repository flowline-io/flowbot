package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestPollingSwapPreservesScroll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		render func() (string, error)
		want   []string
	}{
		{
			name: "hub apps table polling uses show none and preserve scroll",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.HubAppsTable(nil, nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				`hx-swap="outerHTML show:none"`,
				`data-preserve-scroll`,
			},
		},
		{
			name: "healthz status polling uses show none and preserve scroll",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.HealthzStatus(partials.HealthzData{}).Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				`hx-swap="outerHTML show:none"`,
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

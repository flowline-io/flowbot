package layout_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/layout"
)

func TestAuthLayout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "brand and no app nav",
			wantContains: []string{"Flowbot", "Homelab data hub", `data-testid="toast-container"`},
			wantAbsent:   []string{"nav-logout", "nav-group-admin", "nav-group-system"},
		},
		{
			name:         "loads core assets",
			wantContains: []string{"htmx.min.js", "app.js", "daisyui.css"},
			wantAbsent:   []string{"pipeline-editor.js", "chart.js.min.js"},
		},
		{
			name:         "english lang",
			wantContains: []string{`lang="en"`},
			wantAbsent:   []string{"fonts.googleapis.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := layout.Auth("Flowbot — Login").Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in body", w)
				}
			}
			for _, w := range tt.wantAbsent {
				if strings.Contains(body, w) {
					t.Fatalf("did not want %q in body", w)
				}
			}
		})
	}
}

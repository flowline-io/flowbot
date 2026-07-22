package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestAgentResourcePreviewLoadFull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		truncated bool
		fullURL   string
		want      []string
		wantNot   []string
	}{
		{
			name:      "truncated with full url shows load full button",
			truncated: true,
			fullURL:   "/service/web/agent-sessions/s1/resources?uri=file%3A%2F%2Fbig.txt&full=1",
			want: []string{
				`data-testid="agent-resource-load-full"`,
				"Load full",
				`hx-swap="outerHTML"`,
				"full=1",
			},
		},
		{
			name:      "truncated without full url has no button",
			truncated: true,
			fullURL:   "",
			wantNot:   []string{`data-testid="agent-resource-load-full"`},
		},
		{
			name:      "not truncated has no load full button",
			truncated: false,
			fullURL:   "/service/web/agent-sessions/s1/resources?uri=file%3A%2F%2Fx&full=1",
			wantNot:   []string{`data-testid="agent-resource-load-full"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := partials.AgentResourcePreview("note.txt", "<pre>body</pre>", tt.truncated, tt.fullURL).
				Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
			for _, not := range tt.wantNot {
				if strings.Contains(html, not) {
					t.Fatalf("did not want %q in %q", not, html)
				}
			}
		})
	}
}

func TestAgentResourcePreviewURLFull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		uri  string
		full bool
		want string
	}{
		{
			name: "preview without full",
			uri:  "file://note.txt",
			full: false,
			want: "/service/web/agent-sessions/s1/resources?uri=file%3A%2F%2Fnote.txt",
		},
		{
			name: "preview with full",
			uri:  "file://note.txt",
			full: true,
			want: "/service/web/agent-sessions/s1/resources?full=1&uri=file%3A%2F%2Fnote.txt",
		},
		{
			name: "encodes uri query",
			uri:  "file://a b.txt",
			full: false,
			want: "uri=file%3A%2F%2Fa+b.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(partials.AgentResourcePreviewURL("s1", tt.uri, tt.full))
			if tt.name == "encodes uri query" {
				if !strings.Contains(got, tt.want) {
					t.Fatalf("got %q want contain %q", got, tt.want)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

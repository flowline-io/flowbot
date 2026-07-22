package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestCursorLoadMore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		id          string
		listURL     string
		rowsTarget  string
		nextCursor  string
		oob         bool
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:       "initial load more button targets rows with beforeend",
			id:         "agent-sessions-load-more",
			listURL:    "/service/web/agent-sessions/list",
			rowsTarget: "#agent-sessions-rows",
			nextCursor: "42",
			oob:        false,
			wantContain: []string{
				`id="agent-sessions-load-more"`,
				`hx-get="/service/web/agent-sessions/list?cursor=42"`,
				`hx-target="#agent-sessions-rows"`,
				`hx-swap="beforeend"`,
				`data-testid="agent-sessions-load-more"`,
				">Load more</button>",
			},
			wantAbsent: []string{`hx-swap-oob`},
		},
		{
			name:       "oob update keeps load more when more pages remain",
			id:         "agent-sessions-load-more",
			listURL:    "/service/web/agent-sessions/list",
			rowsTarget: "#agent-sessions-rows",
			nextCursor: "7",
			oob:        true,
			wantContain: []string{
				`id="agent-sessions-load-more"`,
				`hx-swap-oob="true"`,
				`hx-get="/service/web/agent-sessions/list?cursor=7"`,
				`hx-target="#agent-sessions-rows"`,
				`hx-swap="beforeend"`,
			},
		},
		{
			name:       "oob deletes load more when cursor is empty",
			id:         "agent-sessions-load-more",
			listURL:    "/service/web/agent-sessions/list",
			rowsTarget: "#agent-sessions-rows",
			nextCursor: "",
			oob:        true,
			wantContain: []string{
				`id="agent-sessions-load-more"`,
				`hx-swap-oob="delete"`,
			},
			wantAbsent: []string{`Load more`, `hx-get=`},
		},
		{
			name:       "empty cursor without oob renders nothing",
			id:         "agent-sessions-load-more",
			listURL:    "/service/web/agent-sessions/list",
			rowsTarget: "#agent-sessions-rows",
			nextCursor: "",
			oob:        false,
			wantAbsent: []string{`agent-sessions-load-more`, `Load more`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := partials.CursorLoadMore(tt.id, tt.listURL, tt.rowsTarget, tt.nextCursor, tt.oob).
				Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(html, absent) {
					t.Fatalf("did not want %q in %q", absent, html)
				}
			}
		})
	}
}

package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestAgentSessionTableLoadMore(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	item := model.AgentSession{
		Flag:      "sess-1",
		Title:     "Demo",
		UID:       "user:a",
		UpdatedAt: now,
		CreatedAt: now,
	}
	tests := []struct {
		name       string
		render     func() (string, error)
		want       []string
		wantAbsent []string
	}{
		{
			name: "full table uses beforeend load more",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.AgentSessionTable([]model.AgentSession{item}, "42").Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				`data-testid="agent-sessions-table"`,
				`id="agent-sessions-rows"`,
				`hx-target="#agent-sessions-rows"`,
				`hx-swap="beforeend"`,
				`hx-get="/service/web/agent-sessions/list?cursor=42"`,
			},
			wantAbsent: []string{`hx-swap-oob`},
		},
		{
			name: "append fragment returns rows without outer table",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.AgentSessionTableAppend([]model.AgentSession{item}, "7").Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				"Demo",
				`hx-swap-oob="true"`,
				`hx-target="#agent-sessions-rows"`,
				`hx-swap="beforeend"`,
				`hx-get="/service/web/agent-sessions/list?cursor=7"`,
			},
			wantAbsent: []string{
				`data-testid="agent-sessions-table"`,
				`id="agent-sessions-rows"`,
			},
		},
		{
			name: "append last page deletes load more via oob",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.AgentSessionTableAppend([]model.AgentSession{item}, "").Render(context.Background(), &buf)
				return buf.String(), err
			},
			want: []string{
				"Demo",
				`id="agent-sessions-load-more"`,
				`hx-swap-oob="delete"`,
			},
			wantAbsent: []string{`Load more`},
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
			for _, absent := range tt.wantAbsent {
				if strings.Contains(html, absent) {
					t.Fatalf("did not want %q in %q", absent, html)
				}
			}
		})
	}
}

package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestChatAgentThreadSessionsDrawer(t *testing.T) {
	t.Parallel()
	session := model.AgentSession{Flag: "sess-1", Title: "Demo", State: "Active"}
	endpoints := ChatAgentEndpoints{
		ListURL:     "/service/web/agents/list",
		MessagesURL: "/service/web/agents/sess-1/messages",
		CancelURL:   "/service/web/agents/sess-1/cancel",
		CloseURL:    "/service/web/agents/sess-1/close",
		EventsURL:   "/service/web/agents/sess-1/events",
	}
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "renders sessions open control for small screens",
			want: []string{
				`data-testid="chatagent-sessions-open"`,
				"flex lg:hidden",
				`aria-label="Sessions"`,
			},
		},
		{
			name: "renders sessions drawer dialog with list hx-get",
			want: []string{
				`data-testid="chatagent-sessions-drawer"`,
				`id="chatagent-sessions-drawer"`,
				`hx-get="/service/web/agents/list"`,
				`data-testid="chatagent-sessions-drawer-body"`,
			},
		},
		{
			name: "drawer body loads when dialog opens",
			want: []string{
				`hx-trigger="intersect once"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentThread(session, nil, nil, endpoints, nil).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
		})
	}
}

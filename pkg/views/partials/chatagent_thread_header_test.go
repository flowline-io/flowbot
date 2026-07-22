package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestChatAgentThreadHeaderMobile(t *testing.T) {
	t.Parallel()
	session := model.AgentSession{
		Flag:          "sess-long-id",
		Title:         "询问日期与目录结构",
		State:         "Active",
		Model:         "deepseek-v4-flash",
		ThinkingLevel: "default",
	}
	endpoints := ChatAgentEndpoints{
		ListURL:     "/service/web/agents/list",
		MessagesURL: "/service/web/agents/sess-long-id/messages",
		CancelURL:   "/service/web/agents/sess-long-id/cancel",
		CloseURL:    "/service/web/agents/sess-long-id/close",
		InspectURL:  "/service/web/agent-sessions/sess-long-id",
		EventsURL:   "/service/web/agents/sess-long-id/events",
	}
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "header uses dedicated class for mobile styling",
			want: []string{
				`data-testid="chatagent-thread-header"`,
				"chatagent-thread-header",
			},
		},
		{
			name: "title truncates on narrow screens",
			want: []string{
				`data-testid="chatagent-thread-title"`,
				"chatagent-thread-title",
			},
		},
		{
			name: "session flag hidden below large screens",
			want: []string{
				`data-testid="chatagent-thread-session-id"`,
				"hidden lg:flex",
			},
		},
		{
			name: "desktop actions stay on larger screens",
			want: []string{
				`data-testid="chatagent-header-actions-desktop"`,
				"hidden lg:flex",
			},
		},
		{
			name: "mobile overflow menu contains session actions",
			want: []string{
				`data-testid="chatagent-header-menu"`,
				`data-testid="chatagent-header-menu-panel"`,
				`id="chatagent-header-actions-dialog"`,
				"flex lg:hidden",
				"Close session",
				"Inspect entries",
				"Back to Agents",
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

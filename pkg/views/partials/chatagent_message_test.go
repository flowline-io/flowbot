package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestChatAgentMessageCopyMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		msg       model.AgentChatMessage
		streaming bool
		wantCopy  bool
		wantMD    string
	}{
		{
			name: "assistant with html offers copy markdown",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				Text: "## Hello\n\n- one",
				HTML: "<h2>Hello</h2><ul><li>one</li></ul>",
			},
			wantCopy: true,
			wantMD:   "## Hello\n\n- one",
		},
		{
			name: "assistant plain text offers copy markdown",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				Text: "plain reply",
			},
			wantCopy: true,
			wantMD:   "plain reply",
		},
		{
			name: "user message has no copy button",
			msg: model.AgentChatMessage{
				Role: "user",
				Kind: "user",
				Text: "hello",
			},
			wantCopy: false,
		},
		{
			name: "streaming assistant hides copy button",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				Text: "partial",
				HTML: "<p>partial</p>",
			},
			streaming: true,
			wantCopy:  false,
		},
		{
			name: "empty assistant text has no copy button",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				HTML: "<p></p>",
			},
			wantCopy: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentMessage(tt.msg, tt.streaming).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			hasCopy := strings.Contains(html, `data-testid="chatagent-copy-md"`)
			if hasCopy != tt.wantCopy {
				t.Fatalf("copy button present=%v want=%v\nhtml=%s", hasCopy, tt.wantCopy, html)
			}
			if !tt.wantCopy {
				return
			}
			if !strings.Contains(html, `data-clip-copy`) {
				t.Fatalf("want data-clip-copy on copy button\nhtml=%s", html)
			}
			if !strings.Contains(html, `aria-label="Copy markdown"`) {
				t.Fatalf("want icon button aria-label\nhtml=%s", html)
			}
			if !strings.Contains(html, `chatagent-message-meta`) {
				t.Fatalf("want meta row inside bubble\nhtml=%s", html)
			}
			if !strings.Contains(html, "<svg") {
				t.Fatalf("want copy icon svg\nhtml=%s", html)
			}
			if !strings.Contains(html, tt.wantMD) {
				t.Fatalf("want markdown %q in data-clip-markdown\nhtml=%s", tt.wantMD, html)
			}
		})
	}
}

func TestChatAgentThreadScriptsIncludesClipCopy(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := ChatAgentThreadScripts().Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "/static/js/clip-copy.js") {
		t.Fatalf("want clip-copy.js in thread scripts\nhtml=%s", html)
	}
}

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
	if !strings.Contains(html, "/static/js/chatagent-codeblocks.js") {
		t.Fatalf("want chatagent-codeblocks.js in thread scripts\nhtml=%s", html)
	}
}

func TestChatAgentToolMessageCollapse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		msg      model.AgentChatMessage
		wantOpen bool
	}{
		{
			name: "completed tool is collapsed details",
			msg: model.AgentChatMessage{
				Kind:       "tool",
				ToolName:   "run_terminal",
				ToolStatus: "completed",
				ToolStdout: "ok",
			},
			wantOpen: false,
		},
		{
			name: "error tool is expanded details",
			msg: model.AgentChatMessage{
				Kind:       "tool",
				ToolName:   "run_terminal",
				ToolStatus: "error",
				ToolStderr: "boom",
			},
			wantOpen: true,
		},
		{
			name: "needs_approval tool is expanded details",
			msg: model.AgentChatMessage{
				Kind:       "tool",
				ToolName:   "write_file",
				ToolStatus: "needs_approval",
			},
			wantOpen: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentToolMessage(tt.msg).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, "chatagent-tool") {
				t.Fatalf("want chatagent-tool details\nhtml=%s", html)
			}
			if !strings.Contains(html, "<details") {
				t.Fatalf("want details element\nhtml=%s", html)
			}
			if !strings.Contains(html, "<summary") {
				t.Fatalf("want summary header\nhtml=%s", html)
			}
			hasOpen := strings.Contains(html, "<details open") ||
				strings.Contains(html, " open ") ||
				strings.Contains(html, " open>")
			if hasOpen != tt.wantOpen {
				t.Fatalf("open attr present=%v want=%v\nhtml=%s", hasOpen, tt.wantOpen, html)
			}
		})
	}
}

func TestChatAgentThinkingDefaultsCollapsed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  model.AgentChatMessage
	}{
		{
			name: "thinking without duration",
			msg: model.AgentChatMessage{
				Kind: "thinking",
				Text: "reason…",
			},
		},
		{
			name: "thinking with duration",
			msg: model.AgentChatMessage{
				Kind:               "thinking",
				Text:               "reason…",
				ThinkingDurationMs: 1200,
			},
		},
		{
			name: "thinking with html body",
			msg: model.AgentChatMessage{
				Kind: "thinking",
				HTML: "<p>reason</p>",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentMessage(tt.msg, false).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, `data-testid="chatagent-message-thinking"`) {
				t.Fatalf("want thinking details\nhtml=%s", html)
			}
			if strings.Contains(html, "<details open") || strings.Contains(html, " open>") {
				t.Fatalf("thinking must default collapsed\nhtml=%s", html)
			}
		})
	}
}

func TestChatAgentThreadJumpToBottomControl(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		messages []model.AgentChatMessage
		pending  *ChatAgentPendingConfirm
	}{
		{name: "empty thread includes jump control"},
		{
			name: "thread with user message includes jump control",
			messages: []model.AgentChatMessage{{
				Role: "user", Kind: "user", Text: "hi",
			}},
		},
		{
			name: "pending approval thread includes jump control",
			pending: &ChatAgentPendingConfirm{
				ID: "c-jump", Tool: "run_terminal", Summary: "ls",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentThread(
				model.AgentSession{Flag: "sess-1", State: "Active"},
				tt.messages,
				nil,
				ChatAgentEndpoints{
					MessagesURL:       "/service/web/agents/sess-1/messages",
					CancelURL:         "/service/web/agents/sess-1/cancel",
					ConfirmURL:        "/service/web/agents/sess-1/confirm",
					EventsURL:         "/service/web/agents/sess-1/events",
					RenderMarkdownURL: "/service/web/agents/render-markdown",
				},
				tt.pending,
			).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, `data-testid="chatagent-jump-bottom"`) {
				t.Fatalf("want jump-to-bottom control\nhtml=%s", html)
			}
			if !strings.Contains(html, `id="chatagent-jump-bottom"`) {
				t.Fatalf("want jump-to-bottom id\nhtml=%s", html)
			}
		})
	}
}

func TestChatAgentThreadPendingApprovalEmptyState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		pending    *ChatAgentPendingConfirm
		messages   []model.AgentChatMessage
		wantWait   bool
		wantOpen   bool
		wantHidden bool
	}{
		{
			name: "pending with empty history shows waiting + open panel",
			pending: &ChatAgentPendingConfirm{
				ID:      "c-1",
				Tool:    "run_terminal",
				Summary: "command: ls",
			},
			wantWait: true,
			wantOpen: true,
		},
		{
			name:       "no pending keeps panel hidden",
			wantHidden: true,
		},
		{
			name: "pending with history still opens panel and keeps waiting copy",
			pending: &ChatAgentPendingConfirm{
				ID:   "c-2",
				Tool: "write_file",
			},
			messages: []model.AgentChatMessage{{
				Role: "user",
				Kind: "user",
				Text: "hello",
			}},
			wantWait: true,
			wantOpen: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := ChatAgentThread(
				model.AgentSession{Flag: "sess-pending", State: "Active"},
				tt.messages,
				nil,
				ChatAgentEndpoints{
					MessagesURL: "/service/web/agents/sess-pending/messages",
					ConfirmURL:  "/service/web/agents/sess-pending/confirm",
					EventsURL:   "/service/web/agents/sess-pending/events",
				},
				tt.pending,
			).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			hasWait := strings.Contains(html, `data-testid="chatagent-run-waiting"`)
			if hasWait != tt.wantWait {
				t.Fatalf("waiting copy present=%v want=%v", hasWait, tt.wantWait)
			}
			hasPending := strings.Contains(html, `data-pending-confirm-id=`)
			if tt.wantOpen && !hasPending {
				t.Fatalf("want pending confirm attrs\nhtml=%s", html)
			}
			if tt.wantHidden && hasPending {
				t.Fatalf("did not want pending confirm attrs\nhtml=%s", html)
			}
			if tt.wantOpen && strings.Contains(html, `chatagent-approval-panel shrink-0 flowbot-surface mx-1 mb-2 hidden`) {
				t.Fatalf("pending panel should not include hidden class\nhtml=%s", html)
			}
		})
	}
}

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
		name         string
		msg          model.AgentChatMessage
		streaming    bool
		wantCopyMD   bool
		wantUserCopy bool
		wantMD       string
	}{
		{
			name: "assistant with html offers copy markdown",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				Text: "## Hello\n\n- one",
				HTML: "<h2>Hello</h2><ul><li>one</li></ul>",
			},
			wantCopyMD: true,
			wantMD:     "## Hello\n\n- one",
		},
		{
			name: "assistant plain text offers copy markdown",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				Text: "plain reply",
			},
			wantCopyMD: true,
			wantMD:     "plain reply",
		},
		{
			name: "user message offers plain-text copy",
			msg: model.AgentChatMessage{
				Role: "user",
				Kind: "user",
				Text: "hello",
			},
			wantUserCopy: true,
			wantMD:       "hello",
		},
		{
			name: "user image attachment renders preview and copy",
			msg: model.AgentChatMessage{
				Role: "user",
				Kind: "user",
				Text: "What is this",
				Attachments: []model.AgentChatAttachment{
					{
						FileID:   "img-1",
						MIMEType: "image/png",
						Kind:     "image",
						URL:      "/service/web/agents/s1/media/img-1",
					},
				},
			},
			wantUserCopy: true,
			wantMD:       "What is this",
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
		},
		{
			name: "empty assistant text has no copy button",
			msg: model.AgentChatMessage{
				Role: "assistant",
				Kind: "assistant",
				HTML: "<p></p>",
			},
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
			assertChatAgentCopyHTML(t, buf.String(), tt.name, tt.wantCopyMD, tt.wantUserCopy, tt.wantMD)
		})
	}
}

func assertChatAgentCopyHTML(t *testing.T, html, name string, wantCopyMD, wantUserCopy bool, wantMD string) {
	t.Helper()
	hasCopyMD := strings.Contains(html, `data-testid="chatagent-copy-md"`)
	if hasCopyMD != wantCopyMD {
		t.Fatalf("copy markdown present=%v want=%v\nhtml=%s", hasCopyMD, wantCopyMD, html)
	}
	hasUserCopy := strings.Contains(html, `data-testid="chatagent-copy-user"`)
	if hasUserCopy != wantUserCopy {
		t.Fatalf("user copy present=%v want=%v\nhtml=%s", hasUserCopy, wantUserCopy, html)
	}
	if name == "user image attachment renders preview and copy" {
		assertChatAgentUserImagePreview(t, html)
	}
	if !wantCopyMD && !wantUserCopy {
		return
	}
	assertChatAgentCopyPayload(t, html, wantCopyMD, wantUserCopy, wantMD)
}

func assertChatAgentUserImagePreview(t *testing.T, html string) {
	t.Helper()
	if !strings.Contains(html, `data-testid="chatagent-message-attach-img"`) {
		t.Fatalf("want image preview\nhtml=%s", html)
	}
	if !strings.Contains(html, `src="/service/web/agents/s1/media/img-1"`) {
		t.Fatalf("want preview src\nhtml=%s", html)
	}
}

func assertChatAgentCopyPayload(t *testing.T, html string, wantCopyMD, wantUserCopy bool, wantMD string) {
	t.Helper()
	if !strings.Contains(html, `data-clip-copy`) {
		t.Fatalf("want data-clip-copy on copy button\nhtml=%s", html)
	}
	if wantCopyMD {
		assertChatAgentMarkdownCopyAttrs(t, html)
	}
	if wantUserCopy {
		assertChatAgentUserCopyAttrs(t, html)
	}
	if !strings.Contains(html, "<svg") {
		t.Fatalf("want copy icon svg\nhtml=%s", html)
	}
	if !strings.Contains(html, wantMD) {
		t.Fatalf("want copy payload %q\nhtml=%s", wantMD, html)
	}
}

func assertChatAgentMarkdownCopyAttrs(t *testing.T, html string) {
	t.Helper()
	if !strings.Contains(html, `aria-label="Copy markdown"`) {
		t.Fatalf("want icon button aria-label\nhtml=%s", html)
	}
	if !strings.Contains(html, `chatagent-message-meta`) {
		t.Fatalf("want meta row under assistant body\nhtml=%s", html)
	}
	if !strings.Contains(html, `data-clip-markdown`) {
		t.Fatalf("want data-clip-markdown on assistant copy\nhtml=%s", html)
	}
}

func assertChatAgentUserCopyAttrs(t *testing.T, html string) {
	t.Helper()
	if strings.Contains(html, `data-clip-markdown`) {
		t.Fatalf("user copy must use data-clip-text, not data-clip-markdown\nhtml=%s", html)
	}
	if !strings.Contains(html, `data-clip-text`) {
		t.Fatalf("want data-clip-text on user copy\nhtml=%s", html)
	}
}

func TestChatAgentUserBubbleChrome(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	row := model.AgentChatMessage{Role: "user", Kind: "user", Text: "hi", TurnDurationMs: 900}
	if err := ChatAgentMessage(row, false).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "chatagent-user-bubble") {
		t.Fatalf("want user bubble class\nhtml=%s", html)
	}
	if strings.Contains(html, "chat-bubble") || strings.Contains(html, "bg-primary") {
		t.Fatalf("user must not use daisyUI primary bubble\nhtml=%s", html)
	}
	if strings.Contains(html, `data-testid="chatagent-message-duration"`) {
		t.Fatalf("user must not show turn duration\nhtml=%s", html)
	}
}

func TestChatAgentAssistantProseChrome(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	row := model.AgentChatMessage{
		Role: "assistant", Kind: "assistant", Text: "done", HTML: "<p>done</p>",
		TurnDurationMs: 400,
	}
	if err := ChatAgentMessage(row, false).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "chatagent-assistant-body") {
		t.Fatalf("want assistant body class\nhtml=%s", html)
	}
	if strings.Contains(html, "chat-bubble") {
		t.Fatalf("assistant must not use chat-bubble\nhtml=%s", html)
	}
	bodyClassIdx := strings.Index(html, `class="chatagent-assistant-body`)
	metaTestIdx := strings.Index(html, `data-testid="chatagent-message-meta"`)
	if bodyClassIdx < 0 || metaTestIdx < 0 || metaTestIdx < bodyClassIdx {
		t.Fatalf("want meta after assistant body\nhtml=%s", html)
	}
	bodyTagStart := strings.LastIndex(html[:bodyClassIdx+1], "<div")
	metaTagStart := strings.LastIndex(html[:metaTestIdx+1], "<div")
	if bodyTagStart < 0 || metaTagStart < 0 || metaTagStart <= bodyTagStart {
		t.Fatalf("want distinct body and meta tags\nhtml=%s", html)
	}
	between := html[bodyTagStart:metaTagStart]
	opens := strings.Count(between, "<div")
	closes := strings.Count(between, "</div>")
	if opens != closes {
		t.Fatalf("meta must be a sibling of the assistant body, not a child (opens=%d closes=%d)\nhtml=%s", opens, closes, html)
	}
}

func TestChatAgentThinkingPreviewFirstLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	row := model.AgentChatMessage{
		Kind: "thinking",
		Text: "The user asks how many dirs.\nMore reasoning.",
	}
	if err := ChatAgentMessage(row, false).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "chatagent-step-label") {
		t.Fatalf("want Think label\nhtml=%s", html)
	}
	previewStart := strings.Index(html, `data-testid="chatagent-step-preview"`)
	bodyStart := strings.Index(html, `data-testid="chatagent-message-body"`)
	if previewStart < 0 || bodyStart < 0 {
		t.Fatalf("want preview and body\nhtml=%s", html)
	}
	previewHTML := html[previewStart:bodyStart]
	if !strings.Contains(previewHTML, "The user asks how many dirs.") {
		t.Fatalf("want first-line preview\npreview=%s", previewHTML)
	}
	if strings.Contains(previewHTML, "More reasoning.") {
		t.Fatalf("preview must be first line only\npreview=%s", previewHTML)
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
	if !strings.Contains(html, "/static/js/chatagent-trajectory.js") {
		t.Fatalf("want chatagent-trajectory.js in thread scripts\nhtml=%s", html)
	}
	trajIdx := strings.Index(html, "/static/js/chatagent-trajectory.js")
	threadIdx := strings.Index(html, "/static/js/chatagent-thread.js")
	if trajIdx < 0 || threadIdx < 0 || trajIdx > threadIdx {
		t.Fatalf("want chatagent-trajectory.js before chatagent-thread.js\nhtml=%s", html)
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
			assertChatAgentToolCollapseHTML(t, buf.String(), tt.msg, tt.wantOpen)
		})
	}
}

func htmlHasOpenDetails(html string) bool {
	return strings.Contains(html, "<details open") ||
		strings.Contains(html, " open ") ||
		strings.Contains(html, " open>")
}

func assertChatAgentToolCollapseHTML(t *testing.T, html string, msg model.AgentChatMessage, wantOpen bool) {
	t.Helper()
	if !strings.Contains(html, "chatagent-tool") {
		t.Fatalf("want chatagent-tool details\nhtml=%s", html)
	}
	if !strings.Contains(html, "<details") {
		t.Fatalf("want details element\nhtml=%s", html)
	}
	if !strings.Contains(html, "<summary") {
		t.Fatalf("want summary header\nhtml=%s", html)
	}
	hasOpen := htmlHasOpenDetails(html)
	if hasOpen != wantOpen {
		t.Fatalf("open attr present=%v want=%v\nhtml=%s", hasOpen, wantOpen, html)
	}
	if strings.Contains(html, "chat-bubble") || strings.Contains(html, "flowbot-chip") {
		t.Fatalf("tool must be a one-line summary, not a bubble/chip\nhtml=%s", html)
	}
	assertChatAgentToolStatusHidden(t, html, msg)
}

func assertChatAgentToolStatusHidden(t *testing.T, html string, msg model.AgentChatMessage) {
	t.Helper()
	if msg.ToolStdout != "" && !strings.Contains(html, `data-testid="chatagent-step-preview"`) {
		t.Fatalf("want stdout preview in summary\nhtml=%s", html)
	}
	if strings.Contains(html, "chatagent-step-status") || strings.Contains(html, `data-testid="chatagent-tool-status"`) {
		t.Fatalf("tool status must not appear in the collapsed one-liner\nhtml=%s", html)
	}
	if msg.ToolStatus != "" && !strings.Contains(html, `data-tool-status="`+msg.ToolStatus+`"`) {
		t.Fatalf("want data-tool-status on details\nhtml=%s", html)
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
			if !strings.Contains(html, "chatagent-step-label") {
				t.Fatalf("want Think label class\nhtml=%s", html)
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

func TestChatAgentThreadTrajectoryView(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := ChatAgentThread(
		model.AgentSession{Flag: "sess-1", State: "Active", Title: "Task"},
		nil,
		nil,
		ChatAgentEndpoints{
			MessagesURL:       "/service/web/agents/sess-1/messages",
			CancelURL:         "/service/web/agents/sess-1/cancel",
			ConfirmURL:        "/service/web/agents/sess-1/confirm",
			EventsURL:         "/service/web/agents/sess-1/events",
			RenderMarkdownURL: "/service/web/agents/render-markdown",
			TrajectoryURL:     "/service/web/agents/sess-1/trajectory",
		},
		nil,
	).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	wants := []string{
		`data-trajectory-url="/service/web/agents/sess-1/trajectory"`,
		`data-testid="chatagent-view-toggle"`,
		`data-testid="chatagent-view-chat"`,
		`data-testid="chatagent-view-trajectory"`,
		`data-testid="chatagent-trajectory"`,
		`data-testid="chatagent-trajectory-gantt"`,
		`data-testid="chatagent-trajectory-inspector"`,
		`data-testid="chatagent-input-bar"`,
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Fatalf("want %q\nhtml=%s", want, html)
		}
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
		wantAlways bool
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
		{
			name: "suggest always shows matching allow copy",
			pending: &ChatAgentPendingConfirm{
				ID:               "c-3",
				Tool:             "run_terminal",
				Summary:          "command: ls",
				SuggestAlways:    true,
				SuggestedPattern: "run_terminal:ls *",
			},
			wantWait:   true,
			wantOpen:   true,
			wantAlways: true,
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
			assertChatAgentPendingApprovalHTML(t, buf.String(), tt.wantWait, tt.wantOpen, tt.wantHidden, tt.wantAlways)
		})
	}
}

func assertChatAgentPendingApprovalHTML(t *testing.T, html string, wantWait, wantOpen, wantHidden, wantAlways bool) {
	t.Helper()
	hasWait := strings.Contains(html, `data-testid="chatagent-run-waiting"`)
	if hasWait != wantWait {
		t.Fatalf("waiting copy present=%v want=%v", hasWait, wantWait)
	}
	hasPending := strings.Contains(html, `data-pending-confirm-id=`)
	if wantOpen && !hasPending {
		t.Fatalf("want pending confirm attrs\nhtml=%s", html)
	}
	if wantHidden && hasPending {
		t.Fatalf("did not want pending confirm attrs\nhtml=%s", html)
	}
	if wantOpen && strings.Contains(html, `chatagent-approval-panel shrink-0 hidden`) {
		t.Fatalf("pending panel should not include hidden class\nhtml=%s", html)
	}
	assertChatAgentPendingApprovalOpen(t, html, wantOpen)
	assertChatAgentPendingApprovalAlways(t, html, wantAlways)
	assertChatAgentPendingApprovalOrder(t, html, wantOpen)
}

func assertChatAgentPendingApprovalOpen(t *testing.T, html string, wantOpen bool) {
	t.Helper()
	if !wantOpen {
		return
	}
	if !strings.Contains(html, `data-testid="chatagent-approve-once"`) || !strings.Contains(html, "Allow once") {
		t.Fatalf("want Allow once button\nhtml=%s", html)
	}
	if !strings.Contains(html, `data-testid="chatagent-approve-once-hint"`) {
		t.Fatalf("want Allow once hint\nhtml=%s", html)
	}
}

func assertChatAgentPendingApprovalAlways(t *testing.T, html string, wantAlways bool) {
	t.Helper()
	if !wantAlways {
		return
	}
	if !strings.Contains(html, "Always allow matching") {
		t.Fatalf("want Always allow matching label\nhtml=%s", html)
	}
	if !strings.Contains(html, "run_terminal:ls *") {
		t.Fatalf("want always pattern hint\nhtml=%s", html)
	}
	if !strings.Contains(html, "chatagent-approval-choice-always") {
		t.Fatalf("want always choice button class\nhtml=%s", html)
	}
}

func assertChatAgentPendingApprovalOrder(t *testing.T, html string, wantOpen bool) {
	t.Helper()
	if !wantOpen {
		return
	}
	approvalIdx := strings.Index(html, `id="chatagent-approval-panel"`)
	inputIdx := strings.Index(html, `data-testid="chatagent-input-bar"`)
	messagesIdx := strings.Index(html, `id="chatagent-messages"`)
	if approvalIdx < 0 || inputIdx < 0 || messagesIdx < 0 {
		t.Fatalf("missing approval/input/messages markers\nhtml=%s", html)
	}
	if !(messagesIdx < approvalIdx && approvalIdx < inputIdx) {
		t.Fatalf("want messages → approval → input order, got messages=%d approval=%d input=%d", messagesIdx, approvalIdx, inputIdx)
	}
}

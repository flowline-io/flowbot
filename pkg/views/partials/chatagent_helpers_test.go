package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChatAgentDetailURL(t *testing.T) {
	tests := []struct {
		name     string
		template string
		id       string
		want     string
	}{
		{
			name:     "replaces id placeholder",
			template: "/service/web/agents/{id}",
			id:       "sess-1",
			want:     "/service/web/agents/sess-1",
		},
		{
			name:     "empty id leaves trailing slash",
			template: "/agents/{id}",
			id:       "",
			want:     "/agents/",
		},
		{
			name:     "no placeholder unchanged",
			template: "/agents/list",
			id:       "sess-1",
			want:     "/agents/list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ChatAgentDetailURL(tt.template, tt.id))
		})
	}
}

func TestChatAgentPendingPromptKey(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		want      string
	}{
		{name: "prefixes session id", sessionID: "abc", want: "flowbot-chatagent-pending:abc"},
		{name: "empty session id", sessionID: "", want: "flowbot-chatagent-pending:"},
		{name: "special chars preserved", sessionID: "s-1/x", want: "flowbot-chatagent-pending:s-1/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ChatAgentPendingPromptKey(tt.sessionID))
		})
	}
}

func TestFormatChatAgentMessageHTML(t *testing.T) {
	tests := []struct {
		name string
		role string
		text string
		want string
	}{
		{name: "empty text", role: "user", text: "  ", want: ""},
		{name: "user escapes html", role: "user", text: "<b>x</b>", want: "<pre"},
		{name: "assistant markdown", role: "assistant", text: "**hi**", want: "<strong>hi</strong>"},
		{name: "assistant strips script", role: "assistant", text: `<script>alert(1)</script>`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatChatAgentMessageHTML(tt.role, tt.text)
			if tt.want == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestEnhanceChatAgentMarkdownHTML(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		wantSubstr []string
		wantSame   bool
	}{
		{
			name:       "wraps table",
			html:       "<table><tr><td>x</td></tr></table>",
			wantSubstr: []string{"chatagent-md-table-wrap", "</table></div>"},
		},
		{
			name:     "skips non table html",
			html:     "<p>hello</p>",
			wantSame: true,
		},
		{
			name:     "empty html",
			html:     "",
			wantSame: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enhanceChatAgentMarkdownHTML(tt.html)
			if tt.wantSame {
				assert.Equal(t, tt.html, got)
				return
			}
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, got, sub)
			}
		})
	}
}

func TestRenderChatAgentMarkdownHTML(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantSubstr []string
		wantEmpty  bool
	}{
		{
			name:       "gfm table",
			text:       "| A | B |\n| --- | --- |\n| 1 | 2 |",
			wantSubstr: []string{"chatagent-md-table-wrap", "<table>", "<th", "A", "1"},
		},
		{
			name:      "empty text",
			text:      "  ",
			wantEmpty: true,
		},
		{
			name:       "inline code",
			text:       "use `ls -all` here",
			wantSubstr: []string{"<code>ls -all</code>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderChatAgentMarkdownHTML(tt.text)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, got, sub)
			}
		})
	}
}

func TestParseToolSummaryLine(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantName string
		wantArgs string
		wantOK   bool
	}{
		{name: "parses tool summary", text: `run_terminal({"command":"ls -all"})`, wantName: "run_terminal", wantArgs: `{"command":"ls -all"}`, wantOK: true},
		{name: "invalid without parens", text: "run_terminal", wantOK: false},
		{name: "invalid with spaces in name", text: "run terminal()", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, ok := ParseToolSummaryLine(tt.text)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestClassifyHistoryMessage(t *testing.T) {
	tests := []struct {
		name      string
		role      string
		text      string
		wantKinds []string
	}{
		{name: "user message", role: "user", text: "hello", wantKinds: []string{"user"}},
		{name: "tool summary", role: "assistant", text: `run_terminal({"command":"ls"})`, wantKinds: []string{"tool"}},
		{name: "assistant prose", role: "assistant", text: "Here is the result.", wantKinds: []string{"assistant"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyHistoryMessage(tt.role, tt.text, time.Time{})
			kinds := make([]string, 0, len(got))
			for _, item := range got {
				kinds = append(kinds, item.Kind)
			}
			assert.Equal(t, tt.wantKinds, kinds)
		})
	}
}

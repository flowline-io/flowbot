package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/model"
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

func TestChatAgentSessionTitle(t *testing.T) {
	tests := []struct {
		name string
		item model.AgentSession
		want string
	}{
		{name: "uses title when set", item: model.AgentSession{Title: "CI fix", Flag: "sess-1"}, want: "CI fix"},
		{name: "falls back to flag", item: model.AgentSession{Flag: "sess-2"}, want: "sess-2"},
		{name: "untitled when empty", item: model.AgentSession{}, want: "Untitled session"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ChatAgentSessionTitle(tt.item))
		})
	}
}

func TestFormatChatAgentRelativeTime(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "zero time", at: time.Time{}, want: ""},
		{name: "minutes ago", at: now.Add(-12 * time.Minute), want: "12m"},
		{name: "hours ago", at: now.Add(-5 * time.Hour), want: "5h"},
		{name: "days ago", at: now.Add(-6 * 24 * time.Hour), want: "6d"},
		{name: "weeks ago", at: now.Add(-14 * 24 * time.Hour), want: "2w"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatAgentRelativeTimeSince(tt.at, now))
		})
	}
}

func TestChatAgentSessionBadgeClass(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{name: "active", state: "Active", want: "agents-session-badge-active"},
		{name: "running", state: "Running", want: "agents-session-badge-running"},
		{name: "needs approval", state: "NeedsApproval", want: "agents-session-badge-needs-approval"},
		{name: "closed", state: "Closed", want: "agents-session-badge-closed"},
		{name: "unknown", state: "Unknown", want: "agents-session-badge-unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, chatAgentSessionBadgeClass(tt.state), tt.want)
		})
	}
}

func TestChatAgentSessionListURL(t *testing.T) {
	tests := []struct {
		name      string
		endpoints ChatAgentEndpoints
		cursor    string
		want      string
	}{
		{
			name:      "default list",
			endpoints: ChatAgentEndpoints{ListURL: "/service/web/agents/list"},
			want:      "/service/web/agents/list",
		},
		{
			name:      "with filter",
			endpoints: ChatAgentEndpoints{ListURL: "/service/web/agents/list", Filter: "running"},
			want:      "/service/web/agents/list?filter=running",
		},
		{
			name:      "with filter and cursor",
			endpoints: ChatAgentEndpoints{ListURL: "/service/web/agents/list", Filter: "archived"},
			cursor:    "12",
			want:      "/service/web/agents/list?filter=archived&cursor=12",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ChatAgentSessionListURL(tt.endpoints, tt.cursor))
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

func TestChatAgentToolCardExpanded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "completed stays collapsed", status: "completed", want: false},
		{name: "running stays collapsed", status: "running", want: false},
		{name: "empty stays collapsed", status: "", want: false},
		{name: "error expands", status: "error", want: true},
		{name: "failed expands", status: "failed", want: true},
		{name: "needs_approval expands", status: "needs_approval", want: true},
		{name: "Needs Approval expands case insensitive", status: "Needs_Approval", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ChatAgentToolCardExpanded(tt.status))
		})
	}
}

func TestRenderChatAgentMarkdownHTML(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantSubstr []string
		wantAbsent []string
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
		{
			name:       "inline math in table",
			text:       "| A | B |\n| --- | --- |\n| $10^0 = 1$ | $\\lg 1 = 0$ |",
			wantSubstr: []string{"chatagent-md-table-wrap", "katex", "katex-html"},
			wantAbsent: []string{"$10^0 = 1$"},
		},
		{
			name:       "external link opens in new tab",
			text:       "See [site](https://example.com) for details.",
			wantSubstr: []string{`href="https://example.com"`, `target="_blank"`, `noopener`},
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
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, got, absent)
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

func TestFormatChatAgentDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{name: "zero", ms: 0, want: ""},
		{name: "milliseconds", ms: 250, want: "250ms"},
		{name: "seconds", ms: 2300, want: "2.3s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ChatAgentDurationLabel(tt.ms))
		})
	}
}

func TestChatAgentSessionSettingsLabel(t *testing.T) {
	tests := []struct {
		name         string
		session      model.AgentSession
		defaultModel string
		want         string
	}{
		{
			name:         "falls back to default model and default thinking",
			session:      model.AgentSession{},
			defaultModel: "gpt-test",
			want:         "gpt-test · Thinking: Default",
		},
		{
			name:         "shows stored model and thinking",
			session:      model.AgentSession{Model: "claude", ThinkingLevel: "high"},
			defaultModel: "gpt-test",
			want:         "claude · Thinking: High",
		},
		{
			name:         "empty everything",
			session:      model.AgentSession{},
			defaultModel: "",
			want:         "Thinking: Default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, chatAgentSessionSettingsLabel(tt.session, tt.defaultModel))
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

func TestTruncateChatAgentSessionPreview(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{name: "empty text", text: "  ", limit: 40, want: ""},
		{name: "short text unchanged", text: "hello world", limit: 40, want: "hello world"},
		{name: "truncates long text with ellipsis", text: "abcdefghijklmnopqrstuvwxyz0123456789", limit: 10, want: "abcdefghi…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TruncateChatAgentSessionPreview(tt.text, tt.limit))
		})
	}
}

func TestGroupAgentSessionsByDay(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	today := time.Date(2026, 7, 22, 10, 0, 0, 0, time.Local)
	yesterday := time.Date(2026, 7, 21, 18, 0, 0, 0, time.Local)
	earlier := time.Date(2026, 7, 10, 9, 0, 0, 0, time.Local)

	tests := []struct {
		name      string
		items     []model.AgentSession
		wantKeys  []string
		wantLens  []int
		wantFirst []string
	}{
		{
			name:     "empty list",
			items:    nil,
			wantKeys: nil,
			wantLens: nil,
		},
		{
			name: "groups today yesterday and older",
			items: []model.AgentSession{
				{Flag: "a", UpdatedAt: today},
				{Flag: "b", UpdatedAt: yesterday},
				{Flag: "c", UpdatedAt: earlier},
				{Flag: "d", UpdatedAt: today.Add(-time.Hour)},
			},
			wantKeys:  []string{"today", "yesterday", "2026-07-10"},
			wantLens:  []int{2, 1, 1},
			wantFirst: []string{"a", "b", "c"},
		},
		{
			name: "pinned section precedes day groups",
			items: []model.AgentSession{
				{Flag: "old-pin", Pinned: true, UpdatedAt: earlier},
				{Flag: "new", UpdatedAt: today},
			},
			wantKeys:  []string{"pinned", "today"},
			wantLens:  []int{1, 1},
			wantFirst: []string{"old-pin", "new"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := GroupAgentSessionsForList(tt.items, now)
			keys := make([]string, 0, len(groups))
			lens := make([]int, 0, len(groups))
			first := make([]string, 0, len(groups))
			for _, g := range groups {
				keys = append(keys, g.Key)
				lens = append(lens, len(g.Items))
				if len(g.Items) > 0 {
					first = append(first, g.Items[0].Flag)
				}
			}
			if tt.wantKeys == nil {
				assert.Empty(t, keys)
				assert.Empty(t, lens)
				return
			}
			assert.Equal(t, tt.wantKeys, keys)
			assert.Equal(t, tt.wantLens, lens)
			if tt.wantFirst != nil {
				assert.Equal(t, tt.wantFirst, first)
			}
		})
	}
}

func TestChatAgentSessionActivityLabel(t *testing.T) {
	tests := []struct {
		name     string
		activity string
		want     string
	}{
		{name: "running", activity: "running", want: "Running"},
		{name: "needs approval", activity: "needs_approval", want: "Needs approval"},
		{name: "empty", activity: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ChatAgentSessionActivityLabel(tt.activity))
		})
	}
}

func TestChatAgentApprovalActionCopy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		suggestedPattern string
		wantOnce         string
		wantOnceHint     string
		wantAlways       string
		wantAlwaysHint   string
		wantDeny         string
	}{
		{
			name:           "without suggested pattern",
			wantOnce:       "Allow once",
			wantOnceHint:   "This tool call only",
			wantAlways:     "Always allow matching",
			wantAlwaysHint: "Remember this pattern for future matching calls",
			wantDeny:       "Deny",
		},
		{
			name:             "with suggested pattern",
			suggestedPattern: "run_terminal:ls *",
			wantOnce:         "Allow once",
			wantOnceHint:     "This tool call only",
			wantAlways:       "Always allow matching",
			wantAlwaysHint:   "Remember for future matching calls: run_terminal:ls *",
			wantDeny:         "Deny",
		},
		{
			name:             "trims suggested pattern whitespace",
			suggestedPattern: "  edit:a.txt  ",
			wantOnce:         "Allow once",
			wantOnceHint:     "This tool call only",
			wantAlways:       "Always allow matching",
			wantAlwaysHint:   "Remember for future matching calls: edit:a.txt",
			wantDeny:         "Deny",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantOnce, ChatAgentApproveOnceLabel())
			assert.Equal(t, tt.wantOnceHint, ChatAgentApproveOnceHint())
			assert.Equal(t, tt.wantAlways, ChatAgentApproveAlwaysLabel())
			assert.Equal(t, tt.wantAlwaysHint, ChatAgentApproveAlwaysHint(tt.suggestedPattern))
			assert.Equal(t, tt.wantDeny, ChatAgentApproveDenyLabel())
		})
	}
}

func TestFormatPendingApprovalBadgeText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{name: "zero hides badge", count: 0, want: ""},
		{name: "negative hides badge", count: -1, want: ""},
		{name: "single digit", count: 3, want: "3"},
		{name: "caps at 99 plus", count: 100, want: "99+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FormatPendingApprovalBadgeText(tt.count))
		})
	}
}

func TestChatAgentModelMultimodal(t *testing.T) {
	t.Parallel()
	models := []SelectableModelOption{
		{ID: "text", Name: "Text", Multimodal: false},
		{ID: "vision", Name: "Vision", Multimodal: true},
	}
	tests := []struct {
		name     string
		models   []SelectableModelOption
		selected string
		want     bool
	}{
		{name: "selected multimodal model", models: models, selected: "vision", want: true},
		{name: "selected text-only model", models: models, selected: "text", want: false},
		{name: "empty selection uses first model", models: models, selected: "", want: false},
		{name: "unknown selection falls back to first", models: models, selected: "missing", want: false},
		{name: "no models", models: nil, selected: "vision", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, chatAgentModelMultimodal(tt.models, tt.selected))
		})
	}
}

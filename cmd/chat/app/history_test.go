package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSystemLine(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "prefixes bullet", text: "Initializing agent...", want: "· Initializing agent..."},
		{name: "plain status", text: "Session ended", want: "· Session ended"},
		{name: "error hint", text: "Permission error: denied", want: "· Permission error: denied"},
	}
	styles := NewStyles()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(FormatSystemLine(tt.text, &styles))
			assert.Equal(t, tt.want, strings.TrimSpace(got))
		})
	}
}

func TestIndentAssistantContinuation(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  []string
	}{
		{name: "single line unchanged", lines: []string{"◆ hello"}, want: []string{"◆ hello"}},
		{name: "indents continuation", lines: []string{"◆ line one", "line two"}, want: []string{"◆ line one", "  line two"}},
		{name: "skips blank lines", lines: []string{"◆ a", "", "b"}, want: []string{"◆ a", "", "  b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, indentAssistantContinuation(tt.lines))
		})
	}
}

func TestFormatHistoryLineRoles(t *testing.T) {
	styles := NewStyles()
	tests := []struct {
		name       string
		role       string
		text       string
		wantSubstr []string
	}{
		{name: "user bullet", role: "user", text: "hello", wantSubstr: []string{"●", "hello"}},
		{name: "assistant bullet", role: "assistant", text: "reply", wantSubstr: []string{"◆", "reply"}},
		{name: "unknown role plain", role: "system", text: "note", wantSubstr: []string{"note"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(FormatHistoryLine(tt.role, tt.text, &styles))
			for _, want := range tt.wantSubstr {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestFormatThinkingBlock(t *testing.T) {
	styles := NewStyles()
	tests := []struct {
		name       string
		text       string
		wantSubstr []string
		wantEmpty  bool
	}{
		{
			name:       "renders thinking marker",
			text:       "planning steps",
			wantSubstr: []string{"Thinking", "planning steps"},
		},
		{
			name:       "trims whitespace",
			text:       "  spaced  ",
			wantSubstr: []string{"Thinking", "spaced"},
		},
		{
			name:      "empty text",
			text:      "   ",
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(FormatThinkingBlock(tt.text, 80, &styles))
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			for _, want := range tt.wantSubstr {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestFormatAssistantBlock(t *testing.T) {
	styles := NewStyles()
	tests := []struct {
		name       string
		text       string
		wantSubstr []string
		wantOnce   string
	}{
		{
			name:       "plain text marker",
			text:       "hello",
			wantSubstr: []string{"◆", "hello"},
			wantOnce:   "◆",
		},
		{
			name:       "multiline marker once",
			text:       "line one\nline two",
			wantSubstr: []string{"◆", "line one", "line two"},
			wantOnce:   "◆",
		},
		{
			name:       "markdown body",
			text:       "# Title\nbody",
			wantSubstr: []string{"◆", "Title", "body"},
			wantOnce:   "◆",
		},
		{
			name:       "tool payload summarized",
			text:       `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{}"}}]`,
			wantSubstr: []string{"◆", `run_code({})`},
			wantOnce:   "◆",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(FormatAssistantBlock(tt.text, 80, &styles))
			if tt.wantSubstr == nil {
				assert.Empty(t, got)
				return
			}
			for _, want := range tt.wantSubstr {
				assert.Contains(t, got, want)
			}
			if tt.wantOnce != "" {
				assert.Equal(t, 1, strings.Count(got, tt.wantOnce))
			}
		})
	}
}

func TestFormatHistoryMessages(t *testing.T) {
	styles := NewStyles()
	tests := []struct {
		name        string
		msgs        []client.ChatHistoryMessage
		wantContain []string
		wantNoGap   bool
		wantTurnSep bool
	}{
		{
			name: "user and agent without separator",
			msgs: []client.ChatHistoryMessage{
				{Role: "user", Text: "hello"},
				{Role: "assistant", Text: "hi there"},
			},
			wantContain: []string{"● hello", "hi there"},
			wantNoGap:   true,
		},
		{
			name: "second turn keeps separator",
			msgs: []client.ChatHistoryMessage{
				{Role: "user", Text: "first"},
				{Role: "assistant", Text: "reply one"},
				{Role: "user", Text: "second"},
				{Role: "assistant", Text: "reply two"},
			},
			wantContain: []string{"● first", "reply one", "● second", "reply two"},
			wantTurnSep: true,
		},
		{
			name: "assistant only has marker",
			msgs: []client.ChatHistoryMessage{
				{Role: "assistant", Text: "welcome"},
			},
			wantContain: []string{"welcome"},
			wantNoGap:   true,
		},
		{
			name: "tool payload summarized in history",
			msgs: []client.ChatHistoryMessage{
				{Role: "user", Text: "run"},
				{Role: "assistant", Text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{}"}}]`},
				{Role: "assistant", Text: "done"},
			},
			wantContain: []string{"● run", "run_code({})", "done"},
		},
		{
			name: "coalesce consecutive tool snapshots",
			msgs: []client.ChatHistoryMessage{
				{Role: "user", Text: "weather"},
				{Role: "assistant", Text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":""}}]`},
				{Role: "assistant", Text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":""}}]`},
				{Role: "assistant", Text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"广州天气\"}"}}]`},
			},
			wantContain: []string{"web_search({\"query\":\"广州天气\"})"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(FormatHistoryMessages(tt.msgs, 80, &styles))
			for _, want := range tt.wantContain {
				assert.Contains(t, got, want)
			}
			if tt.name == "coalesce consecutive tool snapshots" {
				assert.Equal(t, 1, strings.Count(got, "◆"))
			}
			if tt.wantNoGap {
				userIdx := strings.Index(got, "●")
				agentIdx := strings.Index(got, "◆")
				if userIdx >= 0 && agentIdx > userIdx {
					between := got[userIdx:agentIdx]
					assert.NotContains(t, between, "─")
					assert.NotContains(t, between, "\n\n")
				}
			}
			if tt.wantTurnSep {
				assert.GreaterOrEqual(t, strings.Count(got, "─"), 1)
			}
		})
	}
}

func TestEstimateHistoryTokens(t *testing.T) {
	tests := []struct {
		name string
		msgs []client.ChatHistoryMessage
		want int
	}{
		{name: "empty history", msgs: nil, want: 0},
		{name: "single message", msgs: []client.ChatHistoryMessage{{Text: string(make([]byte, 400))}}, want: 100},
		{name: "multiple messages", msgs: []client.ChatHistoryMessage{
			{Text: "hello"},
			{Text: string(make([]byte, 396))},
		}, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EstimateHistoryTokens(tt.msgs))
		})
	}
}

func TestApplyHistoryUsage(t *testing.T) {
	tests := []struct {
		name       string
		tokens     int
		window     int
		wantTokens int
		wantPct    float64
	}{
		{name: "restores usage on resume", tokens: 4016, window: 128000, wantTokens: 4016, wantPct: 3.1375},
		{name: "defaults window", tokens: 6400, window: 0, wantTokens: 6400, wantPct: 5},
		{name: "catalog window on resume", tokens: 83, window: 0, wantTokens: 83, wantPct: 0.007919},
		{name: "clears to zero", tokens: 0, window: 128000, wantTokens: 0, wantPct: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			if tt.name == "catalog window on resume" {
				m.info = &client.ChatAgentInfo{ChatModel: "deepseek-v4-flash"}
			}
			m.status.ContextWindow = tt.window
			m.applyHistoryUsage(tt.tokens)
			assert.Equal(t, tt.wantTokens, m.status.TotalTokens)
			assert.InDelta(t, tt.wantPct, m.status.ContextPercent, 0.0001)
		})
	}
}

func TestResumeSessionID(t *testing.T) {
	tests := []struct {
		name        string
		savedID     string
		listStatus  int
		createID    string
		wantID      string
		wantSaved   string
		wantErr     bool
		wantCreated bool
	}{
		{
			name:        "no saved session creates new",
			createID:    "sess-new",
			wantID:      "sess-new",
			wantSaved:   "sess-new",
			wantCreated: true,
		},
		{
			name:       "valid saved session is reused",
			savedID:    "sess-live",
			listStatus: http.StatusOK,
			wantID:     "sess-live",
			wantSaved:  "sess-live",
		},
		{
			name:        "stale saved session is replaced",
			savedID:     "sess-gone",
			listStatus:  http.StatusNotFound,
			createID:    "sess-fresh",
			wantID:      "sess-fresh",
			wantSaved:   "sess-fresh",
			wantCreated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			if tt.savedID != "" {
				require.NoError(t, SaveSessionID("default", tt.savedID))
			}

			created := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/messages"):
					if tt.listStatus == 0 {
						http.NotFound(w, r)
						return
					}
					w.WriteHeader(tt.listStatus)
					if tt.listStatus == http.StatusOK {
						_, _ = w.Write([]byte(`{"messages":[]}`))
					} else {
						_, _ = w.Write([]byte(`{"error":"not found"}`))
					}
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sessions"):
					created = true
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"session_id":"` + tt.createID + `"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)

			cl := client.NewClient(srv.URL, "token")
			got, err := resumeSessionID(context.Background(), cl, "default")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, got)
			assert.Equal(t, tt.wantCreated, created)

			saved, err := LoadSessionID("default")
			require.NoError(t, err)
			assert.Equal(t, tt.wantSaved, saved)
		})
	}
}

func TestResetSessionUsage(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "clears token counters"},
		{name: "idempotent reset"},
		{name: "safe on fresh model"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.status.TotalTokens = 5000
			m.status.ContextPercent = 12.5
			m.resetSessionUsage()
			assert.Zero(t, m.status.TotalTokens)
			assert.Zero(t, m.status.ContextPercent)
		})
	}
}

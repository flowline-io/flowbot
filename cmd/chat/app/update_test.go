package app

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshStreamingAssistantPreservesOverlay(t *testing.T) {
	tests := []struct {
		name             string
		base             string
		streamingBaseLen int
		assistant        string
		overlay          string
		wantSubstr       []string
	}{
		{
			name:       "keeps tool status after refresh",
			base:       "● user\n\n",
			assistant:  "Hello",
			overlay:    "Running tool: run_terminal...\n",
			wantSubstr: []string{"Hello", "Running tool: run_terminal"},
		},
		{
			name:       "replaces assistant snapshot without duplicating",
			base:       "● user\n\n",
			assistant:  "Hi there",
			overlay:    "Initializing agent...\n",
			wantSubstr: []string{"Hi there", "Initializing agent"},
		},
		{
			name:       "empty overlay still renders assistant",
			base:       "● user\n\n",
			assistant:  "Done",
			overlay:    "",
			wantSubstr: []string{"Done"},
		},
		{
			name:             "stale base length after transcript shrink",
			base:             "",
			streamingBaseLen: 100,
			assistant:        "Recovered",
			overlay:          "Running...\n",
			wantSubstr:       []string{"Recovered", "Running"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseLen := tt.streamingBaseLen
			if baseLen == 0 {
				baseLen = len(tt.base)
			}
			m := &Model{
				phase: PhaseStreaming,
				stream: streamRunState{
					streamingBaseLen: baseLen,
					rawAssistant:     tt.assistant,
				},
				width:  80,
				styles: NewStyles(),
			}
			m.transcript.WriteString(tt.base)
			m.stream.overlay.WriteString(tt.overlay)

			m.refreshStreamingAssistant()
			out := stripANSI(m.transcript.String())
			for _, want := range tt.wantSubstr {
				assert.Contains(t, out, want)
			}
		})
	}
}

func TestHandleSlashHelpUpdatesViewport(t *testing.T) {
	tests := []struct {
		name          string
		splashVisible bool
	}{
		{name: "from splash screen", splashVisible: true},
		{name: "after conversation", splashVisible: false},
		{name: "idle with empty transcript", splashVisible: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			m.splashVisible = tt.splashVisible
			m.syncLayout()

			updated, cmd := m.handleSlash("help", "")
			assert.NotNil(t, cmd)
			assert.False(t, updated.splashVisible)
			assert.Contains(t, updated.transcript.String(), "/new")
			assert.Contains(t, updated.viewport.View(), "/new")
		})
	}
}

func TestAppendUserAfterTranscriptContent(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "first message", text: "hello"},
		{name: "follow-up message", text: "world"},
		{name: "unicode input", text: "你好"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.appendSystem(SlashHelp())

			assert.NotPanics(t, func() {
				m.appendUser(tt.text)
			})
			assert.Contains(t, m.transcript.String(), tt.text)
		})
	}
}

func TestHandleStreamEventDeltaSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		incoming string
		want     string
	}{
		{name: "replaces snapshot", existing: "Hi", incoming: "Hi there", want: "Hi there"},
		{name: "overwrites shorter text", existing: "Hello world", incoming: "Hello", want: "Hello"},
		{name: "empty incoming clears buffer", existing: "Hi", incoming: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{stream: streamRunState{rawAssistant: tt.existing}}
			updated, _ := m.handleStreamEvent(client.ChatStreamEvent{Type: "delta", Text: tt.incoming})
			assert.Equal(t, tt.want, updated.stream.rawAssistant)
		})
	}
}

func TestHandleStreamEventThinkingSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		incoming string
		want     string
	}{
		{name: "replaces thinking snapshot", existing: "old", incoming: "new plan", want: "new plan"},
		{name: "trims whitespace", existing: "x", incoming: "  spaced  ", want: "spaced"},
		{name: "empty incoming clears thinking", existing: "old", incoming: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{stream: streamRunState{rawThinking: tt.existing}}
			updated, _ := m.handleStreamEvent(client.ChatStreamEvent{Type: "thinking", Text: tt.incoming})
			assert.Equal(t, tt.want, updated.stream.rawThinking)
		})
	}
}

func TestRefreshStreamingAssistantRendersThinkingBeforeAnswer(t *testing.T) {
	tests := []struct {
		name      string
		thinking  string
		assistant string
		wantOrder []string
	}{
		{
			name:      "thinking before assistant",
			thinking:  "planning steps",
			assistant: "Final answer",
			wantOrder: []string{"Thinking", "planning steps", "Final answer"},
		},
		{
			name:      "thinking only",
			thinking:  "still thinking",
			assistant: "",
			wantOrder: []string{"Thinking", "still thinking"},
		},
		{
			name:      "assistant only",
			thinking:  "",
			assistant: "Done",
			wantOrder: []string{"Done"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				phase: PhaseStreaming,
				stream: streamRunState{
					streamingBaseLen: 0,
					rawThinking:      tt.thinking,
					rawAssistant:     tt.assistant,
				},
				width:  80,
				styles: NewStyles(),
			}
			m.refreshStreamingAssistant()
			out := stripANSI(m.transcript.String())
			for i, want := range tt.wantOrder {
				idx := strings.Index(out, want)
				assert.GreaterOrEqual(t, idx, 0, "missing %q in %q", want, out)
				if i == 0 {
					continue
				}
				prev := strings.Index(out, tt.wantOrder[i-1])
				assert.Greater(t, idx, prev, "order mismatch for %q after %q", want, tt.wantOrder[i-1])
			}
		})
	}
}

func TestHandleStreamEventCanceledClearsThinking(t *testing.T) {
	tests := []struct {
		name string
		base string
	}{
		{name: "canceled after partial stream", base: "● user\n\n"},
		{name: "canceled with prior transcript", base: "● hello\n\n"},
		{name: "canceled from empty base", base: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				phase: PhaseStreaming,
				stream: streamRunState{
					streamingBaseLen: len(tt.base),
					rawThinking:      "secret plan",
					rawAssistant:     "partial answer",
				},
				width:  80,
				styles: NewStyles(),
			}
			m.transcript.WriteString(tt.base)
			m.refreshStreamingAssistant()
			require.Contains(t, stripANSI(m.transcript.String()), "secret plan")

			updated, _ := m.handleStreamEvent(client.ChatStreamEvent{Type: "canceled"})
			assert.Empty(t, updated.stream.rawThinking)
			assert.Empty(t, updated.stream.rawAssistant)
			out := stripANSI(updated.transcript.String())
			assert.NotContains(t, out, "secret plan")
			assert.NotContains(t, out, "partial answer")
		})
	}
}

func TestHandleStreamEventErrorClearsThinking(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{name: "provider error", message: "upstream failed"},
		{name: "timeout error", message: "deadline exceeded"},
		{name: "generic error", message: "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				phase: PhaseStreaming,
				stream: streamRunState{
					streamingBaseLen: 0,
					rawThinking:      "secret plan",
					rawAssistant:     "partial answer",
				},
				width:  80,
				styles: NewStyles(),
			}
			m.refreshStreamingAssistant()
			require.Contains(t, stripANSI(m.transcript.String()), "secret plan")

			updated, _ := m.handleStreamEvent(client.ChatStreamEvent{Type: "error", Message: tt.message})
			assert.Empty(t, updated.stream.rawThinking)
			assert.Empty(t, updated.stream.rawAssistant)
			out := stripANSI(updated.transcript.String())
			assert.NotContains(t, out, "secret plan")
			assert.NotContains(t, out, "partial answer")
			assert.Contains(t, out, tt.message)
		})
	}
}

func TestFinalizeAssistantClearsThinking(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "clears thinking buffer"},
		{name: "clears assistant buffer"},
		{name: "increments message count"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				phase: PhaseStreaming,
				stream: streamRunState{
					streamingBaseLen: 0,
					rawThinking:      "plan",
					rawAssistant:     "answer",
				},
				width:  80,
				styles: NewStyles(),
			}
			m.finalizeAssistant()
			assert.Empty(t, m.stream.rawThinking)
			assert.Empty(t, m.stream.rawAssistant)
			assert.Equal(t, 1, m.messageCount)
			out := stripANSI(m.transcript.String())
			assert.Contains(t, out, "answer")
			assert.NotContains(t, out, "plan")
		})
	}
}

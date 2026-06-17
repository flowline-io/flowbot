package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
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
				phase:            PhaseStreaming,
				streamingBaseLen: baseLen,
				rawAssistant:     tt.assistant,
				width:            80,
				styles:           NewStyles(),
			}
			m.transcript.WriteString(tt.base)
			m.streamOverlay.WriteString(tt.overlay)

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
			m := &Model{rawAssistant: tt.existing}
			updated, _ := m.handleStreamEvent(client.ChatStreamEvent{Type: "delta", Text: tt.incoming})
			assert.Equal(t, tt.want, updated.rawAssistant)
		})
	}
}

package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestParseSubagentProgress(t *testing.T) {
	tests := []struct {
		name         string
		update       string
		wantSubagent string
		wantTool     string
		wantDetail   string
		wantOK       bool
	}{
		{
			name:         "running tool prefix",
			update:       "[general-purpose] running tool: web_search",
			wantSubagent: "general-purpose",
			wantTool:     "web_search",
			wantOK:       true,
		},
		{
			name:         "tool progress detail",
			update:       "[general-purpose] searching...",
			wantSubagent: "general-purpose",
			wantDetail:   "searching...",
			wantOK:       true,
		},
		{
			name:   "plain tool update",
			update: "searching...",
			wantOK: false,
		},
		{
			name:   "malformed bracket",
			update: "[] running tool: echo",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			subagent, tool, detail, ok := parseSubagentProgress(tt.update)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSubagent, subagent)
			assert.Equal(t, tt.wantTool, tool)
			assert.Equal(t, tt.wantDetail, detail)
		})
	}
}

func TestSubagentToolStatusText(t *testing.T) {
	tests := []struct {
		name     string
		subagent string
		tool     string
		detail   string
		want     string
	}{
		{
			name:     "delegation without tool",
			subagent: "general-purpose",
			want:     "Delegating to subagent: general-purpose...",
		},
		{
			name:     "inner tool start",
			subagent: "general-purpose",
			tool:     "web_search",
			want:     "general-purpose › Running tool: web_search...",
		},
		{
			name:     "inner tool detail",
			subagent: "general-purpose",
			tool:     "web_search",
			detail:   "searching...",
			want:     "general-purpose › web_search: searching...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, subagentToolStatusText(tt.subagent, tt.tool, tt.detail))
		})
	}
}

func TestTaskToolStreamEvent(t *testing.T) {
	tests := []struct {
		name string
		call msg.ToolCallPart
		want StreamEvent
	}{
		{
			name: "annotates subagent",
			call: msg.ToolCallPart{
				Name:      delegateSubagentToolName,
				Arguments: `{"subagent_type":"code-reviewer"}`,
			},
			want: StreamEvent{
				Type:     EventTypeTool,
				Name:     delegateSubagentToolName,
				Subagent: "code-reviewer",
				Status:   "running",
			},
		},
		{
			name: "plain task without subagent",
			call: msg.ToolCallPart{Name: delegateSubagentToolName},
			want: StreamEvent{
				Type:   EventTypeTool,
				Name:   delegateSubagentToolName,
				Status: "running",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, taskToolStreamEvent(tt.call, "running", "", 0))
		})
	}
}

func TestPublishSubagentToolUpdate(t *testing.T) {
	tests := []struct {
		name      string
		updates   []string
		wantNames []string
		wantTools []string
	}{
		{
			name: "tracks inner tool across updates",
			updates: []string{
				"[general-purpose] running tool: web_search",
				"[general-purpose] searching...",
			},
			wantNames: []string{"web_search", "web_search"},
			wantTools: []string{"web_search", "web_search"},
		},
		{
			name: "switches inner tool",
			updates: []string{
				"[general-purpose] running tool: web_search",
				"[general-purpose] running tool: read_file",
			},
			wantNames: []string{"web_search", "read_file"},
			wantTools: []string{"web_search", "read_file"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracker := &apiStreamTracker{coalescer: newStreamCoalescer()}
			var events []StreamEvent
			publisher := recordingPublisher{publish: func(event StreamEvent) error {
				events = append(events, event)
				return nil
			}}
			for _, update := range tt.updates {
				publishSubagentToolUpdate(publisher, tracker, update)
			}
			requireLen := len(tt.wantNames)
			if len(events) < requireLen {
				t.Fatalf("got %d events, want %d", len(events), requireLen)
			}
			for i := range tt.wantNames {
				assert.Equal(t, tt.wantNames[i], events[i].Name)
				assert.Equal(t, "general-purpose", events[i].Subagent)
			}
			assert.Equal(t, tt.wantTools[len(tt.wantTools)-1], tracker.subagentTool)
		})
	}
}

type recordingPublisher struct {
	publish func(StreamEvent) error
}

func (r recordingPublisher) Publish(event StreamEvent) error {
	return r.publish(event)
}

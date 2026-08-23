package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentScheduledTaskStateOptions(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "includes active and paused",
			want: []string{"active", "paused"},
		},
		{
			name: "includes terminal states",
			want: []string{"cancelled", "completed", "failed", "missed"},
		},
		{
			name: "returns six lifecycle states",
			want: []string{"active", "paused", "cancelled", "completed", "failed", "missed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgentScheduledTaskStateOptions()
			for _, state := range tt.want {
				assert.Contains(t, got, state)
			}
			if len(tt.want) == 6 {
				assert.Len(t, got, 6)
			}
		})
	}
}

func TestAgentScheduledTaskStateURL(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
		want   string
	}{
		{name: "builds state endpoint", taskID: "task-123", want: "/service/web/agent-scheduled-tasks/task-123/state"},
		{name: "preserves task id segments", taskID: "task/with/slash", want: "/service/web/agent-scheduled-tasks/task/with/slash/state"},
		{name: "handles empty task id", taskID: "", want: "/service/web/agent-scheduled-tasks//state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(AgentScheduledTaskStateURL(tt.taskID)))
		})
	}
}

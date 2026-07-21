package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestChatAgentSessionTodoLineLabel(t *testing.T) {
	tests := []struct {
		name    string
		summary model.AgentTodoSummary
		want    string
	}{
		{
			name:    "completed session",
			summary: model.AgentTodoSummary{Total: 4, Done: 4, Active: 0},
			want:    "4/4 done",
		},
		{
			name:    "in progress with current step",
			summary: model.AgentTodoSummary{Total: 5, Done: 1, Active: 4, InProgress: "步骤二：设计方案制定"},
			want:    "20% · 4 active · 1/5 · 步骤二：设计方案制定",
		},
		{
			name:    "partial progress without current step",
			summary: model.AgentTodoSummary{Total: 3, Done: 1, Active: 2},
			want:    "33% · 2 active · 1/3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatAgentSessionTodoLineLabel(tt.summary))
		})
	}
}

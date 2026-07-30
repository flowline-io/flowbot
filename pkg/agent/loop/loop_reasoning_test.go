package loop

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectAssistantToolReasoning(t *testing.T) {
	t.Parallel()
	messages := []msg.AgentMessage{
		NewUserMessage("hi"),
		msg.AssistantMessage{
			ThinkingText: "plain think",
			Parts:        []msg.ContentPart{msg.TextPart{Text: "hello"}},
		},
		msg.AssistantMessage{
			ThinkingText: "need skill",
			Parts: []msg.ContentPart{
				msg.ToolCallPart{ID: "1", Name: "read_skill", Arguments: `{"name":"gitea"}`},
				msg.ToolCallPart{ID: "2", Name: "echo", Arguments: `{}`},
			},
		},
		msg.ToolResultMessage{ToolCallID: "1", Name: "read_skill", Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}}},
		msg.AssistantMessage{
			Parts: []msg.ContentPart{
				msg.ToolCallPart{ID: "3", Name: "echo", Arguments: `{}`},
			},
		},
	}
	assert.Equal(t, map[string]string{
		"1": "need skill",
		"2": "need skill",
		"3": "",
	}, collectAssistantToolReasoning(messages))
}

func TestWithEnsuredToolCallIDs(t *testing.T) {
	t.Parallel()
	messages := withEnsuredToolCallIDs([]msg.AgentMessage{
		msg.AssistantMessage{
			ThinkingText: "plan",
			Parts: []msg.ContentPart{
				msg.ToolCallPart{ID: "", Name: "echo", Arguments: `{}`},
				msg.ToolCallPart{ID: "keep", Name: "echo", Arguments: `{}`},
			},
		},
	})
	assistant, ok := messages[0].(msg.AssistantMessage)
	require.True(t, ok)
	calls := assistant.ToolCalls()
	require.Len(t, calls, 2)
	assert.NotEmpty(t, calls[0].ID)
	assert.Greater(t, len(calls[0].ID), 5)
	assert.Equal(t, "keep", calls[1].ID)
	assert.Equal(t, map[string]string{
		calls[0].ID: "plan",
		"keep":      "plan",
	}, collectAssistantToolReasoning(messages))
}

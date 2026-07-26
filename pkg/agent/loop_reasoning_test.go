package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectAssistantToolReasoning(t *testing.T) {
	t.Parallel()
	messages := []AgentMessage{
		NewUserMessage("hi"),
		AssistantMessage{
			ThinkingText: "plain think",
			Parts:        []ContentPart{TextPart{Text: "hello"}},
		},
		AssistantMessage{
			ThinkingText: "need skill",
			Parts: []ContentPart{
				ToolCallPart{ID: "1", Name: "read_skill", Arguments: `{"name":"gitea"}`},
				ToolCallPart{ID: "2", Name: "echo", Arguments: `{}`},
			},
		},
		ToolResultMessage{ToolCallID: "1", Name: "read_skill", Parts: []ContentPart{TextPart{Text: "ok"}}},
		AssistantMessage{
			Parts: []ContentPart{
				ToolCallPart{ID: "3", Name: "echo", Arguments: `{}`},
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
	messages := withEnsuredToolCallIDs([]AgentMessage{
		AssistantMessage{
			ThinkingText: "plan",
			Parts: []ContentPart{
				ToolCallPart{ID: "", Name: "echo", Arguments: `{}`},
				ToolCallPart{ID: "keep", Name: "echo", Arguments: `{}`},
			},
		},
	})
	assistant, ok := messages[0].(AssistantMessage)
	require.True(t, ok)
	calls := assistant.ToolCalls()
	require.Len(t, calls, 2)
	assert.NotEmpty(t, calls[0].ID)
	assert.Greater(t, len(calls[0].ID), 5)
	assert.Equal(t, "keep", calls[1].ID)
	assert.Equal(t, map[string]string{
		calls[0].ID: "plan",
		"keep":       "plan",
	}, collectAssistantToolReasoning(messages))
}

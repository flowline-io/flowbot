package chatagent

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleTrajectory(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)

	trace := session.TreeEntry{
		ID:         "tr1",
		Type:       session.EntryTurnTrace,
		AssembleMs: 9,
		Sections: []session.TraceSection{
			session.NewTraceSection(TraceSectionSystemBody, "identity"),
			session.NewTraceSection(TraceSectionRuntime, "cwd=/tmp"),
		},
	}
	user := session.TreeEntry{
		ID:      "u1",
		Type:    session.EntryMessage,
		Message: msg.NewUserMessage("how many dirs"),
	}
	assistant := session.TreeEntry{
		ID:   "a1",
		Type: session.EntryMessage,
		Message: msg.AssistantMessage{
			Parts: []msg.ContentPart{
				msg.TextPart{Text: "counting"},
				msg.ToolCallPart{ID: "c1", Name: "run_terminal", Arguments: `{"cmd":"ls"}`},
			},
			ThinkingText:       "need to list",
			ThinkingDurationMs: 40,
			TurnDurationMs:     120,
		},
	}
	tool := session.TreeEntry{
		ID:   "t1",
		Type: session.EntryMessage,
		Message: msg.ToolResultMessage{
			ToolCallID: "c1",
			Name:       "run_terminal",
			Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
			DurationMs: 30,
		},
	}
	delegate := session.TreeEntry{
		ID:   "a2",
		Type: session.EntryMessage,
		Message: msg.AssistantMessage{
			Parts: []msg.ContentPart{
				msg.ToolCallPart{ID: "d1", Name: delegateSubagentToolName, Arguments: `{"subagent_type":"explore","prompt":"look"}`},
			},
		},
	}
	delegateResult := session.TreeEntry{
		ID:   "t2",
		Type: session.EntryMessage,
		Message: msg.ToolResultMessage{
			ToolCallID: "d1",
			Name:       delegateSubagentToolName,
			Parts:      []msg.ContentPart{msg.TextPart{Text: "found"}},
		},
	}

	view := assembleTrajectory([]session.TreeEntry{trace, user, assistant, tool, delegate, delegateResult}, map[string]time.Time{
		"tr1": created, "u1": created, "a1": created, "t1": created, "a2": created, "t2": created,
	})
	require.NotNil(t, view)
	kinds := make([]string, 0, len(view.Rows))
	for _, row := range view.Rows {
		kinds = append(kinds, row.Kind)
		assert.NotEmpty(t, row.ID)
	}
	assert.Equal(t, []string{"system", "context", "user", "thinking", "assistant", "tool_call", "tool", "tool_call", "tool"}, kinds)
	assert.Equal(t, "tr1/system_body", view.Rows[0].ID)
	assert.Equal(t, "tr1/runtime", view.Rows[1].ID)
	assert.Equal(t, "identity", view.Rows[0].Text)
	assert.Equal(t, int64(9), view.Rows[0].AssembleMs)
	assert.Equal(t, "run_terminal", view.Rows[5].ToolName)
	assert.Contains(t, view.Rows[5].Text, "ls")
	assert.Equal(t, "explore", view.Rows[7].Subagent)
	assert.Equal(t, "explore", view.Rows[8].Subagent)

	legacy := assembleTrajectory([]session.TreeEntry{user}, map[string]time.Time{"u1": created})
	require.Len(t, legacy.Rows, 1)
	assert.Equal(t, "user", legacy.Rows[0].Kind)
}

func TestAppendTurnTraceSkipsPipeline(t *testing.T) {
	t.Parallel()
	err := appendTurnTrace(t.Context(), nil, RunRequest{Kind: RunKindPipeline}, promptTrace{
		Sections: []session.TraceSection{session.NewTraceSection(TraceSectionSystemBody, "x")},
	})
	assert.NoError(t, err)
}

func TestAppendTurnTracePersistsAndPublishes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := session.NewMemoryStorage()
	sess := session.New(store)
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "root", Type: session.EntryMessage, Message: msg.NewUserMessage("hi"),
	}))
	h := harness.New(harness.Options{Session: sess})
	pub := &apiEventRecorder{}
	sections := []session.TraceSection{session.NewTraceSection(TraceSectionSystemBody, "identity")}
	err := appendTurnTrace(ctx, h, RunRequest{
		Kind:      RunKindInteractive,
		SessionID: "sess-1",
		API:       &APIRunOptions{Publisher: pub},
	}, promptTrace{Sections: sections, AssembleMs: 7})
	require.NoError(t, err)

	branch, err := sess.GetBranch(ctx, "")
	require.NoError(t, err)
	require.Len(t, branch, 2)
	assert.Equal(t, session.EntryTurnTrace, branch[1].Type)
	assert.Equal(t, "root", branch[1].ParentID)
	assert.Equal(t, int64(7), branch[1].AssembleMs)

	events := pub.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypeTurnTrace, events[0].Type)
	assert.Equal(t, branch[1].ID, events[0].ID)
	assert.Equal(t, int64(7), events[0].AssembleMs)
	require.Len(t, events[0].Sections, 1)
	assert.Equal(t, TraceSectionSystemBody, events[0].Sections[0].Name)
}

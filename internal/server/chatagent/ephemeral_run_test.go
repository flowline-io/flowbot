package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAutonomousRunKind(t *testing.T) {
	tests := []struct {
		name string
		kind RunKind
		want bool
	}{
		{name: "scheduled is autonomous", kind: RunKindScheduled, want: true},
		{name: "pipeline is autonomous", kind: RunKindPipeline, want: true},
		{name: "interactive is not autonomous", kind: RunKindInteractive, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsAutonomousRunKind(tt.kind))
		})
	}
}

func TestBeginEphemeralSessionCreatesActiveSession(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())

	setupEphemeralRunTestDB(t)

	sessionID, err := BeginEphemeralSession(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)

	sess, err := store.Database.GetChatSession(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Equal(t, "user-1", sess.UID)

	CloseEphemeralSession(context.Background(), NewService(), sessionID)
}

func TestRunEphemeralReturnsReplyAndClosesSession(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "pipeline agent reply")

	svc := NewService()
	out, err := RunEphemeral(context.Background(), svc, EphemeralRunParams{
		UID:    "user-1",
		Prompt: "summarize",
		Kind:   RunKindPipeline,
	})
	require.NoError(t, err)
	assert.Equal(t, "pipeline agent reply", out.Reply)
	assert.NotEmpty(t, out.SessionID)
	WaitForSessionTitleGenerationForTest()
}

func TestRunAutonomousPromptRejectsEmptyMessage(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	sessionID, err := BeginEphemeralSession(context.Background(), "user-1")
	require.NoError(t, err)
	defer CloseEphemeralSession(context.Background(), NewService(), sessionID)

	_, err = RunAutonomousPrompt(context.Background(), NewService(), sessionID, "   ", RunKindScheduled, nil, nil, "")
	assert.Error(t, err)
}

func TestRunEphemeralReturnsSessionIDOnPromptFailure(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	out, err := RunEphemeral(context.Background(), NewService(), EphemeralRunParams{
		UID:    "user-1",
		Prompt: "   ",
		Kind:   RunKindPipeline,
	})
	require.Error(t, err)
	assert.NotEmpty(t, out.SessionID)
}

package chatagent

import (
	"context"
	"testing"

	abilityagent "github.com/flowline-io/flowbot/pkg/capability/agent"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPipelineAgentUsesEphemeralSession(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "pipeline reply text")

	out, err := RunPipelineAgent(context.Background(), abilityagent.RunParams{
		Prompt: "analyze event",
		UID:    types.Uid("user-9"),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "pipeline reply text", out.Reply)
	assert.NotEmpty(t, out.SessionID)
	WaitForSessionTitleGenerationForTest()
}

func TestRunPipelineAgentRejectsEmptyPrompt(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	_, err := RunPipelineAgent(context.Background(), abilityagent.RunParams{
		Prompt: "   ",
		UID:    types.Uid("user-9"),
	})
	assert.Error(t, err)
}

func TestRunPipelineAgentReturnsSessionIDOnFailure(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	out, err := RunPipelineAgent(context.Background(), abilityagent.RunParams{
		Prompt: "   ",
		UID:    types.Uid("user-9"),
	})
	require.Error(t, err)
	require.NotNil(t, out)
	assert.NotEmpty(t, out.SessionID)
}

func TestPipelineAgentRunnerForwardsToRunPipelineAgent(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "runner reply")

	out, err := PipelineAgentRunner{}.Run(context.Background(), abilityagent.RunParams{
		Prompt: "hello",
		UID:    types.Uid("user-1"),
	})
	require.NoError(t, err)
	assert.Equal(t, "runner reply", out.Reply)
	WaitForSessionTitleGenerationForTest()
}

func TestValidatePipelineAgentTools(t *testing.T) {
	tests := []struct {
		name    string
		tools   []string
		wantErr bool
	}{
		{name: "empty tools allowed", tools: nil},
		{name: "known tool allowed", tools: []string{"read_file"}},
		{name: "unknown tool rejected", tools: []string{"evil_tool"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePipelineAgentTools(tt.tools)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

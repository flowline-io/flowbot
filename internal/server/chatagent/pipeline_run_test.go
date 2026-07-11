package chatagent

import (
	"context"
	"testing"

	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPipelineAgentUsesEphemeralSession(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "pipeline reply text")

	out, err := RunPipelineAgent(context.Background(), "analyze event", types.Uid("user-9"))
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

	_, err := RunPipelineAgent(context.Background(), "   ", types.Uid("user-9"))
	assert.Error(t, err)
}

func TestRunPipelineAgentReturnsSessionIDOnFailure(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	out, err := RunPipelineAgent(context.Background(), "   ", types.Uid("user-9"))
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

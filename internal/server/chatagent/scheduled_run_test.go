package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func setupEphemeralRunTestDB(t *testing.T) store.Adapter {
	t.Helper()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() {
		WaitForSessionTitleGenerationForTest()
		store.Database = origDB
	})
	return store.Database
}

func setupEphemeralRunFakeModel(t *testing.T, reply string) {
	t.Helper()
	ws := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{
		Workspace: ws,
		ChatModel: "fake-model",
		ToolModel: "",
	}
	config.App.Models = []config.Model{
		{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
	}
	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: reply})
	origNewModel := NewModelForTest
	NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
		return model, "fake-model", nil
	}
	t.Cleanup(func() {
		WaitForSessionTitleGenerationForTest()
		ResetSessionTitleGenerationForTest()
		NewModelForTest = origNewModel
	})
}

func TestExecuteScheduledTaskRunsPrompt(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "scheduled task done")

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-run-1",
		UID:          "user:alice",
		Name:         "daily",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "check inbox",
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	ExecuteScheduledTaskForTest(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-run-1",
		UID:          "user:alice",
		Name:         "daily",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "check inbox",
		State:        string(schema.ChatScheduledTaskStateActive),
	})
	WaitForSessionTitleGenerationForTest()

	runs, err := store.Database.ListChatScheduledTaskRuns(context.Background(), "task-run-1", 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, string(schema.ChatScheduledTaskRunStateCompleted), runs[0].State)
	assert.Equal(t, "scheduled task done", runs[0].Reply)
	assert.NotEmpty(t, runs[0].RunSessionID)
	require.NotNil(t, runs[0].FinishedAt)
	assert.WithinDuration(t, time.Now().UTC(), *runs[0].FinishedAt, 5*time.Second)
}

func TestExecuteScheduledTaskMarksFailedOnEmptyPrompt(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-run-fail",
		UID:          "user:alice",
		Name:         "empty",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "   ",
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	ExecuteScheduledTaskForTest(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-run-fail",
		UID:          "user:alice",
		Name:         "empty",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "   ",
		State:        string(schema.ChatScheduledTaskStateActive),
	})

	runs, err := store.Database.ListChatScheduledTaskRuns(context.Background(), "task-run-fail", 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, string(schema.ChatScheduledTaskRunStateFailed), runs[0].State)
	assert.NotEmpty(t, runs[0].Error)
}

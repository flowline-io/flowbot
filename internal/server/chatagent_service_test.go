package server

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestChatAgentService_Run(t *testing.T) {
	ws := t.TempDir()
	config.App.ChatAgent.Workspace = ws
	config.App.Agents = []config.Agent{
		{Name: "chat", Enabled: true, Model: "fake-model"},
	}
	config.App.Models = []config.Model{
		{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
	}

	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "assistant reply"})
	origNewModel := chatagent.NewModelForTest
	chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
		return model, "fake-model", nil
	}
	t.Cleanup(func() { chatagent.NewModelForTest = origNewModel })

	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{}
	testChatSessionEntries = map[string][]*gen.ChatSessionEntry{}
	t.Cleanup(func() {
		store.Database = origDB
		testChatSessions = map[string]*gen.ChatSession{}
		testChatSessionEntries = map[string][]*gen.ChatSessionEntry{}
	})

	require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
		Flag: "sess-1", UID: "u1", State: int(schema.ChatSessionActive),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{name: "successful reply", text: "hello", wantErr: false},
		{name: "empty message", text: "  ", wantErr: true},
		{name: "follow up question", text: "explain more", wantErr: false},
	}

	svc := chatagent.NewService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Run(context.Background(), chatagent.RunRequest{
				SessionID: "sess-1",
				Text:      tt.text,
			})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}

func TestChatAgentService_RunRequiresWorkspace(t *testing.T) {
	config.App.ChatAgent.Workspace = ""
	config.App.Agents = []config.Agent{
		{Name: "chat", Enabled: true, Model: "fake-model"},
	}
	config.App.Models = []config.Model{
		{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
	}

	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testChatSessions = map[string]*gen.ChatSession{
		"sess-1": {Flag: "sess-1", State: int(schema.ChatSessionActive)},
	}
	t.Cleanup(func() {
		store.Database = origDB
		testChatSessions = map[string]*gen.ChatSession{}
	})

	_, err := chatagent.NewService().Run(context.Background(), chatagent.RunRequest{
		SessionID: "sess-1",
		Text:      "hello",
	})
	assert.Error(t, err)
}

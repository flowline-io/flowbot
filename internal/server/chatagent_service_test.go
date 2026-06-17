package server

import (
	"context"
	"strings"
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
	config.App.ChatAgent.ChatModel = "fake-model"
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
			}, nil)
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
	config.App.ChatAgent.ChatModel = "fake-model"
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
	}, nil)
	assert.Error(t, err)
}

func TestChatAgentService_CompactSession(t *testing.T) {
	ws := t.TempDir()
	config.App.ChatAgent = config.ChatAgentConfig{
		Workspace: ws,
		ChatModel: "fake-model",
		Compaction: config.CompactionConfig{
			Auto:     new(false),
			Prune:    new(true),
			Reserved: 100,
		},
	}
	config.App.Models = []config.Model{
		{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
	}

	model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nTest summary"})
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
		chatagent.ResetHarnessPoolForTest()
	})

	require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
		Flag: "sess-compact", UID: "u1", State: int(schema.ChatSessionActive),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	svc := chatagent.NewService()
	_, err := svc.Run(context.Background(), chatagent.RunRequest{
		SessionID: "sess-compact",
		Text:      strings.Repeat("word ", 5000),
	}, nil)
	require.NoError(t, err)

	tests := []struct {
		name          string
		sessionID     string
		wantErr       bool
		wantCompacted bool
	}{
		{name: "compacts existing history", sessionID: "sess-compact", wantCompacted: true},
		{name: "reports missing session", sessionID: "missing", wantErr: true},
		{name: "recompacts compacted leaf", sessionID: "sess-compact", wantCompacted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CompactSession(context.Background(), tt.sessionID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantCompacted, result.Compacted)
			if tt.wantCompacted && tt.name == "compacts existing history" {
				assert.Greater(t, result.TokensBefore, result.TokensAfter)
			}
		})
	}
}

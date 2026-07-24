package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	agentmodel "github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetSessionSettings(t *testing.T) {
	LockAppConfigForTest(t)
	origDB := store.Database
	origCfg := config.App
	store.Database = postgres.NewSQLiteTestAdapter(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-test", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		config.App = origCfg
	})

	ctx := context.Background()
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-settings", UID: "u1", State: int(schema.ChatSessionActive),
	}))

	tests := []struct {
		name    string
		in      SessionSettings
		wantErr error
		want    SessionSettings
	}{
		{
			name: "persists registered model and thinking level",
			in:   SessionSettings{Model: "gpt-alt", ThinkingLevel: "high"},
			want: SessionSettings{Model: "gpt-alt", ThinkingLevel: "high"},
		},
		{
			name:    "rejects unknown model",
			in:      SessionSettings{Model: "nope", ThinkingLevel: "default"},
			wantErr: types.ErrInvalidArgument,
		},
		{
			name:    "rejects invalid thinking level",
			in:      SessionSettings{Model: "gpt-test", ThinkingLevel: "turbo"},
			wantErr: types.ErrInvalidArgument,
		},
		{
			name: "allows empty model to clear override",
			in:   SessionSettings{Model: "", ThinkingLevel: "low"},
			want: SessionSettings{Model: "", ThinkingLevel: "low"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetSessionSettings(ctx, "sess-settings", tt.in)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			got, err := GetSessionSettings(ctx, "sess-settings")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveEffectiveSessionSettings(t *testing.T) {
	LockAppConfigForTest(t)
	origDB := store.Database
	origCfg := config.App
	store.Database = postgres.NewSQLiteTestAdapter(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-default", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		config.App = origCfg
	})

	ctx := context.Background()
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-empty", UID: "u1", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-override", UID: "u1", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, SetSessionSettings(ctx, "sess-override", SessionSettings{
		Model: "gpt-alt", ThinkingLevel: "medium",
	}))

	tests := []struct {
		name          string
		sessionID     string
		wantModel     string
		wantThinking  string
		wantStoredMod string
	}{
		{
			name:         "empty fields fall back to yaml defaults",
			sessionID:    "sess-empty",
			wantModel:    "gpt-default",
			wantThinking: agentllm.ThinkingLevelDefault,
		},
		{
			name:          "uses stored overrides",
			sessionID:     "sess-override",
			wantModel:     "gpt-alt",
			wantThinking:  "medium",
			wantStoredMod: "gpt-alt",
		},
		{
			name:         "missing session falls back to yaml",
			sessionID:    "missing",
			wantModel:    "gpt-default",
			wantThinking: agentllm.ThinkingLevelDefault,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEffectiveSessionSettings(ctx, tt.sessionID)
			assert.Equal(t, tt.wantModel, got.Model)
			assert.Equal(t, tt.wantThinking, got.ThinkingLevel)
			assert.Equal(t, tt.wantStoredMod, got.Stored.Model)
		})
	}
}

func TestBuildSelectableModels(t *testing.T) {
	LockAppConfigForTest(t)
	orig := config.App
	t.Cleanup(func() { config.App = orig })

	tests := []struct {
		name string
		cfg  config.Type
		want []string
	}{
		{
			name: "lists all registered models without dual filter",
			cfg: config.Type{
				ChatAgent: config.ChatAgentConfig{ChatModel: "a"},
				Models: []config.Model{
					{Provider: "openai", ModelNames: []string{"a", "b"}},
					{Provider: "anthropic", ModelNames: []string{"c"}},
				},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "dual model filters to chat provider",
			cfg: config.Type{
				ChatAgent: config.ChatAgentConfig{ChatModel: "a", ToolModel: "b"},
				Models: []config.Model{
					{Provider: "openai", ModelNames: []string{"a", "b"}},
					{Provider: "anthropic", ModelNames: []string{"c"}},
				},
			},
			want: []string{"a", "b"},
		},
		{
			name: "deduplicates model names",
			cfg: config.Type{
				ChatAgent: config.ChatAgentConfig{ChatModel: "a"},
				Models: []config.Model{
					{Provider: "openai", ModelNames: []string{"a"}},
					{Provider: "openai_compatible", ModelNames: []string{"a", "d"}},
				},
			},
			want: []string{"a", "d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App = tt.cfg
			got := BuildSelectableModels()
			ids := make([]string, len(got))
			for i, m := range got {
				ids[i] = m.ID
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestBuildSelectableModelsMultimodal(t *testing.T) {
	LockAppConfigForTest(t)
	orig := config.App
	t.Cleanup(func() { config.App = orig })

	agentmodel.RegisterTestMetadata(t, agentmodel.Metadata{
		ID: "vision-model",
		Features: []agentmodel.Feature{
			agentmodel.ModalityImageIn,
			agentmodel.ModalityTextIn,
			agentmodel.ModalityTextOut,
		},
	})
	agentmodel.RegisterTestMetadata(t, agentmodel.Metadata{
		ID: "text-only-model",
		Features: []agentmodel.Feature{
			agentmodel.ModalityTextIn,
			agentmodel.ModalityTextOut,
		},
	})
	agentmodel.RegisterTestMetadata(t, agentmodel.Metadata{
		ID: "audio-model",
		Features: []agentmodel.Feature{
			agentmodel.ModalityAudioIn,
			agentmodel.ModalityTextIn,
			agentmodel.ModalityTextOut,
		},
	})

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "image input model is multimodal", id: "vision-model", want: true},
		{name: "text-only model is not multimodal", id: "text-only-model", want: false},
		{name: "audio input model is multimodal", id: "audio-model", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App = config.Type{
				ChatAgent: config.ChatAgentConfig{ChatModel: tt.id},
				Models: []config.Model{
					{Provider: "openai", ModelNames: []string{tt.id}},
				},
			}
			got := BuildSelectableModels()
			require.Len(t, got, 1)
			assert.Equal(t, tt.want, got[0].Multimodal)
		})
	}
}

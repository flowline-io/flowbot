package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerDefaultsSaveLoadAndEffective(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{
			ChatModel: "gpt-default",
			ToolModel: "gpt-tool",
		},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-tool", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	require.NoError(t, SaveServerDefaults(ctx, ServerDefaultsFormInput{
		ChatModel:     "gpt-alt",
		ToolModel:     ServerDefaultFormInherit,
		ThinkingLevel: "high",
	}))

	stored, err := LoadServerDefaults(ctx)
	require.NoError(t, err)
	assert.True(t, stored.ChatModelSet)
	assert.Equal(t, "gpt-alt", stored.ChatModel)
	assert.False(t, stored.ToolModelSet)
	assert.True(t, stored.ThinkingLevelSet)
	assert.Equal(t, "high", stored.ThinkingLevel)

	assert.Equal(t, "gpt-alt", EffectiveChatAgentChatModel(ctx))
	chat, tool, dual, err := ResolveEffectiveChatAgentModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, "gpt-alt", chat)
	assert.Equal(t, "gpt-tool", tool)
	assert.True(t, dual)

	level, ok := effectiveServerThinkingLevel(ctx)
	assert.True(t, ok)
	assert.Equal(t, "high", level)
}

func TestServerDefaultsExplicitEmptyToolModel(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{
			ChatModel: "gpt-default",
			ToolModel: "gpt-tool",
		},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-tool"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	require.NoError(t, SaveServerDefaults(ctx, ServerDefaultsFormInput{
		ChatModel: ServerDefaultFormInherit,
		ToolModel: ServerDefaultToolNone,
		ThinkingLevel: ServerDefaultFormInherit,
	}))

	_, tool, dual, err := ResolveEffectiveChatAgentModels(ctx)
	require.NoError(t, err)
	assert.Empty(t, tool)
	assert.False(t, dual)
}

func TestServerDefaultsDelete(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-default"},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	require.NoError(t, SaveServerDefaults(ctx, ServerDefaultsFormInput{
		ChatModel: "gpt-alt",
	}))
	require.NoError(t, DeleteServerDefaults(ctx))

	assert.Equal(t, "gpt-default", EffectiveChatAgentChatModel(ctx))
	stored, err := LoadServerDefaults(ctx)
	require.NoError(t, err)
	assert.False(t, stored.anySet())
}

func TestServerDefaultsValidation(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-default"},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	err := SaveServerDefaults(ctx, ServerDefaultsFormInput{ChatModel: "missing"})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidArgument)
}

func TestResolveEffectiveSessionSettingsUsesServerDefaults(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-default", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-alt"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	require.NoError(t, SaveServerDefaults(ctx, ServerDefaultsFormInput{
		ChatModel:     ServerDefaultFormInherit,
		ThinkingLevel: "low",
	}))
	require.NoError(t, store.ChatStoreFromDB().CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-server-default", UID: "u1", State: int(schema.ChatSessionActive),
	}))

	got := ResolveEffectiveSessionSettings(ctx, "sess-server-default")
	assert.Equal(t, "gpt-default", got.Model)
	assert.Equal(t, "low", got.ThinkingLevel)
	assert.Empty(t, got.Stored.ThinkingLevel)
}

func TestResolveEffectiveSessionSettingsKeepsSessionModelOverride(t *testing.T) {
	LockAppConfigForTest(t)
	origCfg := config.App
	installSQLiteTestDatabase(t)
	config.App = config.Type{
		ChatAgent: config.ChatAgentConfig{ChatModel: "gpt-default", Workspace: t.TempDir()},
		Models: []config.Model{
			{Provider: "openai", ApiKey: "k", ModelNames: []string{"gpt-default", "gpt-alt", "gpt-session"}},
		},
	}
	t.Cleanup(func() {
		config.App = origCfg
		ResetServerDefaultsCacheForTest()
	})

	ctx := context.Background()
	require.NoError(t, store.ChatStoreFromDB().CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-override", UID: "u1", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, SetSessionSettings(ctx, "sess-override", SessionSettings{
		Model: "gpt-session", ThinkingLevel: "high",
	}))
	require.NoError(t, SaveServerDefaults(ctx, ServerDefaultsFormInput{
		ChatModel:     "gpt-alt",
		ThinkingLevel: "low",
	}))

	got := ResolveEffectiveSessionSettings(ctx, "sess-override")
	assert.Equal(t, "gpt-session", got.Model)
	assert.Equal(t, "high", got.ThinkingLevel)
}

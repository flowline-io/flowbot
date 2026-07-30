package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/config"
)

func setupSQLiteTestDB(t *testing.T) {
	t.Helper()
	orig := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = orig })
}

func setupChatAgentHTTPTest(t *testing.T, sessions ...*gen.ChatSession) {
	t.Helper()
	origCfg := config.App.ChatAgent
	setupSQLiteTestDB(t)
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: t.TempDir()}
	for _, sess := range sessions {
		seedTestChatSession(t, sess)
	}
	t.Cleanup(func() {
		config.App.ChatAgent = origCfg
	})
}

func seedTestChatSession(t *testing.T, sess *gen.ChatSession) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	if sess.UID == "" {
		sess.UID = "test-user"
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = now
	}
	require.NoError(t, store.ChatStoreFromDB().CreateChatSession(ctx, sess))
	if sess.LeafID != "" {
		require.NoError(t, store.ChatStoreFromDB().UpdateChatSessionLeaf(ctx, sess.Flag, sess.LeafID))
	}
	if sess.Mode != "" {
		require.NoError(t, store.ChatStoreFromDB().UpdateChatSessionMode(ctx, sess.Flag, sess.Mode))
	}
	if sess.Title != "" {
		require.NoError(t, store.ChatStoreFromDB().UpdateChatSessionTitle(ctx, sess.Flag, sess.Title))
	}
}

func seedTestChatSessionEntry(t *testing.T, entry *gen.ChatSessionEntry) {
	t.Helper()
	require.NoError(t, store.ChatStoreFromDB().CreateChatSessionEntry(context.Background(), entry))
}

func seedTestAgentSkill(t *testing.T, skill *gen.AgentSkill) {
	t.Helper()
	if skill.Content == "" {
		skill.Content = " "
	}
	if skill.Name == "" {
		skill.Name = skill.Flag
	}
	if skill.Description == "" {
		skill.Description = " "
	}
	require.NoError(t, store.AgentStoreFromDB().CreateAgentSkill(context.Background(), skill))
}

func seedTestAgentSkillFile(t *testing.T, file *gen.AgentSkillFile) {
	t.Helper()
	require.NoError(t, store.AgentStoreFromDB().CreateAgentSkillFile(context.Background(), file))
}

func seedTestBot(t *testing.T, bot *gen.Bot) {
	t.Helper()
	_, err := store.PlatformStoreFromDB().CreateBot(context.Background(), bot)
	require.NoError(t, err)
}

func seedTestPlatform(t *testing.T, name string) int64 {
	t.Helper()
	now := time.Now()
	id, err := store.PlatformStoreFromDB().CreatePlatform(context.Background(), &gen.Platform{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	return id
}

func seedTestUser(t *testing.T, flag string) *gen.User {
	t.Helper()
	now := time.Now()
	user := &gen.User{
		Flag:      flag,
		Name:      flag,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, store.UserStoreFromDB().UserCreate(context.Background(), user))
	return user
}

func seedTestPlatformUser(t *testing.T, platformID, userID int64, flag, email, avatarURL string) *gen.PlatformUser {
	t.Helper()
	now := time.Now()
	item := &gen.PlatformUser{
		PlatformID: platformID,
		UserID:     userID,
		Flag:       flag,
		Name:       flag,
		Email:      email,
		AvatarURL:  avatarURL,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := store.UserStoreFromDB().CreatePlatformUser(context.Background(), item)
	require.NoError(t, err)
	got, err := store.UserStoreFromDB().GetPlatformUserByFlag(context.Background(), flag)
	require.NoError(t, err)
	return got
}

func seedTestChannel(t *testing.T, flag string) int64 {
	t.Helper()
	now := time.Now()
	ch := &gen.Channel{
		Name:      flag,
		Flag:      flag,
		CreatedAt: now,
		UpdatedAt: now,
	}
	id, err := store.PlatformStoreFromDB().CreateChannel(context.Background(), ch)
	require.NoError(t, err)
	return id
}

func seedTestAgentPlan(t *testing.T, plan *gen.AgentPlan) {
	t.Helper()
	if plan.Content == "" {
		plan.Content = " "
	}
	if plan.Title == "" {
		plan.Title = " "
	}
	require.NoError(t, store.AgentStoreFromDB().CreateAgentPlan(context.Background(), plan))
}

func seedTestAgentTodos(t *testing.T, sessionID string, items ...*gen.AgentTodo) {
	t.Helper()
	require.NoError(t, store.AgentStoreFromDB().ReplaceAgentTodosForSession(context.Background(), sessionID, items))
}

func getTestChatSession(t *testing.T, flag string) *gen.ChatSession {
	t.Helper()
	sess, err := store.ChatStoreFromDB().GetChatSession(context.Background(), flag)
	require.NoError(t, err)
	return sess
}

func seedTestPlatformChannel(t *testing.T, platformID, channelID int64, flag string) *gen.PlatformChannel {
	t.Helper()
	now := time.Now()
	_, err := store.PlatformStoreFromDB().CreatePlatformChannel(context.Background(), &gen.PlatformChannel{
		PlatformID: platformID,
		ChannelID:  channelID,
		Flag:       flag,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	require.NoError(t, err)
	got, err := store.PlatformStoreFromDB().GetPlatformChannelByFlag(context.Background(), flag)
	require.NoError(t, err)
	return got
}

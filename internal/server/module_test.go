package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// testModuleHandler implements module.Handler for testing registerModules.
type testModuleHandler struct {
	module.Base
	ready bool
}

func (h *testModuleHandler) IsReady() bool { return h.ready }
func (*testModuleHandler) Init(_ json.RawMessage) error {
	return nil
}
func (*testModuleHandler) Register() error  { return nil }
func (*testModuleHandler) Bootstrap() error { return nil }
func (*testModuleHandler) Rules() []any     { return nil }
func (*testModuleHandler) Command(_ types.Context, _ any) (types.MsgPayload, error) {
	return nil, nil
}

// testStoreAdapter satisfies store.Adapter for testing registerModules.
type testStoreAdapter struct {
	bots        map[string]*gen.Bot
	createCalls int
	updateCalls int
}

var (
	testChatSessions       = make(map[string]*gen.ChatSession)
	testChatSessionEntries = make(map[string][]*gen.ChatSessionEntry)
	testAgentSkills        = make(map[string]*gen.AgentSkill)
)

func newTestStoreAdapter() *testStoreAdapter {
	return &testStoreAdapter{
		bots: map[string]*gen.Bot{
			"stale-bot": {Name: "stale-bot", State: int(schema.BotActive)},
		},
	}
}

func (a *testStoreAdapter) GetBotByName(_ context.Context, n string) (*gen.Bot, error) {
	b, _ := a.bots[n]
	return b, nil
}
func (a *testStoreAdapter) CreateBot(_ context.Context, b *gen.Bot) (int64, error) {
	a.createCalls++
	a.bots[b.Name] = b
	return int64(len(a.bots)), nil
}
func (a *testStoreAdapter) UpdateBot(_ context.Context, b *gen.Bot) error {
	a.updateCalls++
	a.bots[b.Name] = b
	return nil
}
func (a *testStoreAdapter) GetBots(_ context.Context) ([]*gen.Bot, error) {
	var list []*gen.Bot
	for _, b := range a.bots {
		list = append(list, b)
	}
	return list, nil
}

func (*testStoreAdapter) Open(config.StoreType) error                   { return nil }
func (*testStoreAdapter) Close() error                                  { return nil }
func (*testStoreAdapter) IsOpen() bool                                  { return true }
func (*testStoreAdapter) GetName() string                               { return "test" }
func (*testStoreAdapter) Stats() any                                    { return nil }
func (*testStoreAdapter) Ping(_ context.Context) (time.Duration, error) { return 0, nil }
func (*testStoreAdapter) GetDB() any                                    { return nil }
func (*testStoreAdapter) UserCreate(_ context.Context, user *gen.User) error {
	user.ID = 100
	return nil
}
func (*testStoreAdapter) UserGet(context.Context, types.Uid) (*gen.User, error) {
	return nil, nil
}
func (*testStoreAdapter) UserGetAll(context.Context, ...types.Uid) ([]*gen.User, error) {
	return nil, nil
}
func (*testStoreAdapter) FirstUser(context.Context) (*gen.User, error) {
	return nil, nil
}
func (*testStoreAdapter) UserDelete(context.Context, types.Uid, bool) error { return nil }
func (*testStoreAdapter) UserUpdate(context.Context, types.Uid, types.KV) error {
	return nil
}
func (*testStoreAdapter) FileStartUpload(context.Context, *types.FileDef) error {
	return nil
}
func (*testStoreAdapter) FileFinishUpload(context.Context, *types.FileDef, bool, int64) (*types.FileDef, error) {
	return nil, nil
}
func (*testStoreAdapter) FileGet(context.Context, string) (*types.FileDef, error) {
	return nil, nil
}
func (*testStoreAdapter) FileDeleteUnused(context.Context, time.Time, int) ([]string, error) {
	return nil, nil
}
func (*testStoreAdapter) GetUsers(context.Context) ([]*gen.User, error) { return nil, nil }
func (*testStoreAdapter) GetUserById(context.Context, int64) (*gen.User, error) {
	return nil, nil
}
func (*testStoreAdapter) GetUserByFlag(context.Context, string) (*gen.User, error) {
	return nil, nil
}
func (*testStoreAdapter) CreatePlatformUser(context.Context, *gen.PlatformUser) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) GetPlatformUsersByUserId(context.Context, int64) ([]*gen.PlatformUser, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatformUserByFlag(context.Context, string) (*gen.PlatformUser, error) {
	return nil, nil
}
func (*testStoreAdapter) UpdatePlatformUser(context.Context, *gen.PlatformUser) error {
	return nil
}
func (*testStoreAdapter) GetPlatformChannelByFlag(context.Context, string) (*gen.PlatformChannel, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatformChannelsByPlatformIds(context.Context, []int64) ([]*gen.PlatformChannel, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatformChannelsByChannelId(context.Context, int64) (*gen.PlatformChannel, error) {
	return nil, nil
}
func (*testStoreAdapter) CreatePlatformChannel(context.Context, *gen.PlatformChannel) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) UpdatePlatformChannelChannelID(context.Context, int64, int64) error {
	return nil
}
func (*testStoreAdapter) CreatePlatformChannelUser(context.Context, *gen.PlatformChannelUser) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) GetPlatformChannelUsersByUserFlag(context.Context, string) ([]*gen.PlatformChannelUser, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatformChannelUsersByUserFlags(context.Context, []string) ([]*gen.PlatformChannelUser, error) {
	return nil, nil
}
func (*testStoreAdapter) GetMessage(context.Context, string) (*gen.Message, error) {
	return nil, nil
}
func (*testStoreAdapter) GetMessageByPlatform(context.Context, int64, string) (*gen.Message, error) {
	return nil, nil
}
func (*testStoreAdapter) GetMessagesBySession(context.Context, string) ([]*gen.Message, error) {
	return nil, nil
}
func (*testStoreAdapter) CreateMessage(context.Context, gen.Message) error { return nil }
func (*testStoreAdapter) CreateChatSession(_ context.Context, session *gen.ChatSession) error {
	testChatSessions[session.Flag] = session
	return nil
}
func (*testStoreAdapter) GetChatSession(_ context.Context, flag string) (*gen.ChatSession, error) {
	sess, ok := testChatSessions[flag]
	if !ok {
		return nil, types.ErrNotFound
	}
	return sess, nil
}
func (*testStoreAdapter) UpdateChatSessionLeaf(_ context.Context, flag, leafID string) error {
	sess, ok := testChatSessions[flag]
	if !ok {
		return types.ErrNotFound
	}
	sess.LeafID = leafID
	return nil
}
func (*testStoreAdapter) CloseChatSession(_ context.Context, flag string) error {
	sess, ok := testChatSessions[flag]
	if !ok {
		return types.ErrNotFound
	}
	sess.State = 2
	return nil
}
func (*testStoreAdapter) CreateChatSessionEntry(_ context.Context, entry *gen.ChatSessionEntry) error {
	testChatSessionEntries[entry.SessionID] = append(testChatSessionEntries[entry.SessionID], entry)
	return nil
}
func (*testStoreAdapter) AppendChatSessionEntry(_ context.Context, entry *gen.ChatSessionEntry) error {
	testChatSessionEntries[entry.SessionID] = append(testChatSessionEntries[entry.SessionID], entry)
	if sess, ok := testChatSessions[entry.SessionID]; ok {
		sess.LeafID = entry.Flag
	}
	return nil
}
func (*testStoreAdapter) ListChatSessionEntries(_ context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	return append([]*gen.ChatSessionEntry(nil), testChatSessionEntries[sessionID]...), nil
}
func (*testStoreAdapter) GetChatSessionEntry(_ context.Context, flag string) (*gen.ChatSessionEntry, error) {
	for _, rows := range testChatSessionEntries {
		for _, row := range rows {
			if row.Flag == flag {
				return row, nil
			}
		}
	}
	return nil, types.ErrNotFound
}
func (*testStoreAdapter) GetChatSessionEntryInSession(_ context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error) {
	for _, row := range testChatSessionEntries[sessionID] {
		if row.Flag == flag {
			return row, nil
		}
	}
	return nil, types.ErrNotFound
}
func (*testStoreAdapter) ListAgentSkills(_ context.Context, enabledOnly bool) ([]*gen.AgentSkill, error) {
	rows := make([]*gen.AgentSkill, 0, len(testAgentSkills))
	for _, skill := range testAgentSkills {
		if enabledOnly && !skill.Enabled {
			continue
		}
		rows = append(rows, skill)
	}
	return rows, nil
}
func (*testStoreAdapter) GetAgentSkillByName(_ context.Context, name string) (*gen.AgentSkill, error) {
	skill, ok := testAgentSkills[name]
	if !ok || !skill.Enabled {
		return nil, types.ErrNotFound
	}
	return skill, nil
}
func (*testStoreAdapter) GetAgentSkillsMaxUpdatedAt(_ context.Context) (time.Time, error) {
	var maxUpdated time.Time
	for _, skill := range testAgentSkills {
		if !skill.Enabled {
			continue
		}
		if skill.UpdatedAt.After(maxUpdated) {
			maxUpdated = skill.UpdatedAt
		}
	}
	return maxUpdated, nil
}
func (*testStoreAdapter) CreateAgentSkill(_ context.Context, skill *gen.AgentSkill) error {
	testAgentSkills[skill.Name] = skill
	return nil
}
func (*testStoreAdapter) UpdateAgentSkill(_ context.Context, skill *gen.AgentSkill) error {
	testAgentSkills[skill.Name] = skill
	return nil
}
func (*testStoreAdapter) GetAgentSkillByFlag(_ context.Context, flag string) (*gen.AgentSkill, error) {
	for _, skill := range testAgentSkills {
		if skill.Flag == flag || (skill.Flag == "" && skill.Name == flag) {
			return skill, nil
		}
	}
	return nil, types.ErrNotFound
}
func (*testStoreAdapter) DeleteAgentSkill(_ context.Context, flag string) error {
	for name, skill := range testAgentSkills {
		if skill.Flag == flag || (skill.Flag == "" && skill.Name == flag) {
			delete(testAgentSkills, name)
			return nil
		}
	}
	return types.ErrNotFound
}
func (*testStoreAdapter) GetBot(context.Context, int64) (*gen.Bot, error) {
	return nil, nil
}
func (*testStoreAdapter) DeleteBot(context.Context, string) error { return nil }
func (*testStoreAdapter) GetPlatform(context.Context, int64) (*gen.Platform, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatformByName(context.Context, string) (*gen.Platform, error) {
	return nil, nil
}
func (*testStoreAdapter) GetPlatforms(context.Context) ([]*gen.Platform, error) {
	return nil, nil
}
func (*testStoreAdapter) CreatePlatform(context.Context, *gen.Platform) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) GetChannel(context.Context, int64) (*gen.Channel, error) {
	return nil, nil
}
func (*testStoreAdapter) GetChannelByName(context.Context, string) (*gen.Channel, error) {
	return nil, nil
}
func (*testStoreAdapter) CreateChannel(_ context.Context, channel *gen.Channel) (int64, error) {
	channel.ID = 100
	return channel.ID, nil
}
func (*testStoreAdapter) UpdateChannel(context.Context, *gen.Channel) error { return nil }
func (*testStoreAdapter) DeleteChannel(context.Context, string) error       { return nil }
func (*testStoreAdapter) GetChannels(context.Context) ([]*gen.Channel, error) {
	return nil, nil
}
func (*testStoreAdapter) DataSet(context.Context, types.Uid, string, string, types.KV) error {
	return nil
}
func (*testStoreAdapter) DataGet(context.Context, types.Uid, string, string) (types.KV, error) {
	return nil, nil
}
func (*testStoreAdapter) DataList(context.Context, types.Uid, string, types.DataFilter) ([]*gen.Data, error) {
	return nil, nil
}
func (*testStoreAdapter) DataDelete(context.Context, types.Uid, string, string) error {
	return nil
}
func (*testStoreAdapter) ConfigSet(context.Context, types.Uid, string, string, types.KV) error {
	return nil
}
func (*testStoreAdapter) ConfigGet(context.Context, types.Uid, string, string) (types.KV, error) {
	return nil, nil
}
func (*testStoreAdapter) ListConfigByPrefix(context.Context, types.Uid, string, string) ([]*gen.ConfigData, error) {
	return nil, nil
}
func (*testStoreAdapter) ConfigDelete(context.Context, types.Uid, string, string) error {
	return nil
}
func (*testStoreAdapter) ListConfigs(_ context.Context, _ store.ListConfigOptions) ([]model.ConfigItem, error) {
	return nil, nil
}
func (*testStoreAdapter) OAuthSet(context.Context, gen.OAuth) error { return nil }
func (*testStoreAdapter) OAuthGet(context.Context, types.Uid, string, string) (gen.OAuth, error) {
	return gen.OAuth{}, nil
}
func (*testStoreAdapter) OAuthGetAvailable(context.Context, string) ([]gen.OAuth, error) {
	return nil, nil
}
func (*testStoreAdapter) FormSet(context.Context, string, gen.Form) error { return nil }
func (*testStoreAdapter) FormGet(context.Context, string) (gen.Form, error) {
	return gen.Form{}, nil
}
func (*testStoreAdapter) PageSet(context.Context, string, gen.Page) error { return nil }
func (*testStoreAdapter) PageGet(context.Context, string) (gen.Page, error) {
	return gen.Page{}, nil
}
func (*testStoreAdapter) BehaviorSet(context.Context, gen.Behavior) error { return nil }
func (*testStoreAdapter) BehaviorGet(context.Context, types.Uid, string) (gen.Behavior, error) {
	return gen.Behavior{}, nil
}
func (*testStoreAdapter) BehaviorList(context.Context, types.Uid) ([]*gen.Behavior, error) {
	return nil, nil
}
func (*testStoreAdapter) BehaviorIncrease(context.Context, types.Uid, string, int) error {
	return nil
}
func (*testStoreAdapter) ParameterSet(context.Context, string, types.KV, time.Time) error {
	return nil
}
func (*testStoreAdapter) ParameterGet(context.Context, string) (gen.Parameter, error) {
	return gen.Parameter{}, nil
}
func (*testStoreAdapter) ParameterDelete(context.Context, string) error { return nil }
func (*testStoreAdapter) CreateInstruct(context.Context, *gen.Instruct) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) ListInstruct(context.Context, types.Uid, bool, int) ([]*gen.Instruct, error) {
	return nil, nil
}
func (*testStoreAdapter) UpdateInstruct(context.Context, *gen.Instruct) error { return nil }
func (*testStoreAdapter) CreateCounter(context.Context, *gen.Counter) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) IncreaseCounter(context.Context, int64, int64) error { return nil }
func (*testStoreAdapter) DecreaseCounter(context.Context, int64, int64) error { return nil }
func (*testStoreAdapter) ListCounter(context.Context, types.Uid, string) ([]*gen.Counter, error) {
	return nil, nil
}
func (*testStoreAdapter) GetCounter(context.Context, int64) (gen.Counter, error) {
	return gen.Counter{}, nil
}
func (*testStoreAdapter) GetCounterByFlag(context.Context, types.Uid, string, string) (gen.Counter, error) {
	return gen.Counter{}, nil
}
func (*testStoreAdapter) GetAgents(context.Context) ([]*gen.Agent, error) { return nil, nil }
func (*testStoreAdapter) GetAgentByHostid(context.Context, types.Uid, string, string) (*gen.Agent, error) {
	return nil, nil
}
func (*testStoreAdapter) CreateAgent(context.Context, *gen.Agent) (int64, error) { return 0, nil }
func (*testStoreAdapter) UpdateAgentLastOnlineAt(context.Context, types.Uid, string, string, time.Time) error {
	return nil
}
func (*testStoreAdapter) UpdateAgentOnlineDuration(context.Context, types.Uid, string, string, time.Time) error {
	return nil
}
func (*testStoreAdapter) CreateNotifyChannel(context.Context, string, string, string) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) GetNotifyChannel(context.Context, int64) (model.NotifyChannel, error) {
	return model.NotifyChannel{}, nil
}
func (*testStoreAdapter) GetNotifyChannelRaw(context.Context, int64) (model.NotifyChannel, error) {
	return model.NotifyChannel{}, nil
}
func (*testStoreAdapter) ListNotifyChannels(context.Context, store.ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	return nil, nil
}
func (*testStoreAdapter) UpdateNotifyChannel(context.Context, int64, string, string, string, bool) error {
	return nil
}
func (*testStoreAdapter) DeleteNotifyChannel(context.Context, int64) error { return nil }
func (*testStoreAdapter) CreateNotifyRule(context.Context, model.NotifyRule) (int64, error) {
	return 0, nil
}
func (*testStoreAdapter) GetNotifyRule(context.Context, int64) (model.NotifyRule, error) {
	return model.NotifyRule{}, nil
}
func (*testStoreAdapter) ListNotifyRules(context.Context, store.ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	return nil, nil
}
func (*testStoreAdapter) UpdateNotifyRule(context.Context, int64, model.NotifyRule) error {
	return nil
}
func (*testStoreAdapter) DeleteNotifyRule(context.Context, int64) error { return nil }
func (*testStoreAdapter) MaskNotifyURI(string, string) string           { return "" }
func (*testStoreAdapter) CreateToken(_ context.Context, _ types.Uid, _ time.Time, _ []string) (string, error) {
	return "", nil
}
func (*testStoreAdapter) ListTokens(_ context.Context) ([]model.TokenItem, error) { return nil, nil }
func (*testStoreAdapter) RevokeToken(_ context.Context, _ string) error           { return nil }

func TestRegisterModules_CreatesNewBot(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates new bot when not found in database"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newTestStoreAdapter()
			originalDB := store.Database
			store.Database = mock
			t.Cleanup(func() { store.Database = originalDB })

			module.Register("test-create-mod-bot-001", &testModuleHandler{ready: false})
			t.Cleanup(func() { module.Unregister("test-create-mod-bot-001") })
			registerModules()

			bot, err := mock.GetBotByName(context.Background(), "test-create-mod-bot-001")
			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, "test-create-mod-bot-001", bot.Name)
			assert.Equal(t, int(schema.BotInactive), bot.State)
		})
	}
}

func TestRegisterModules_DeactivatesStaleBot(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deactivates bots for unregistered modules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newTestStoreAdapter()
			originalDB := store.Database
			store.Database = mock
			t.Cleanup(func() { store.Database = originalDB })

			registerModules()

			bot, err := mock.GetBotByName(context.Background(), "stale-bot")
			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, int(schema.BotInactive), bot.State)
		})
	}
}

func TestRegisterModules_SetsActiveForReadyModule(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "sets active state for ready modules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newTestStoreAdapter()
			originalDB := store.Database
			store.Database = mock
			t.Cleanup(func() { store.Database = originalDB })

			module.Register("test-ready-mod-bot-002", &testModuleHandler{ready: true})
			t.Cleanup(func() { module.Unregister("test-ready-mod-bot-002") })
			registerModules()

			bot, err := mock.GetBotByName(context.Background(), "test-ready-mod-bot-002")
			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, int(schema.BotActive), bot.State)
		})
	}
}

func TestRegisterModules_UpdatesExistingBotState(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "updates existing bot state from inactive to active when module becomes ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newTestStoreAdapter()
			mock.bots["existing-ready-bot"] = &gen.Bot{
				Name:  "existing-ready-bot",
				State: int(schema.BotInactive),
			}
			originalDB := store.Database
			store.Database = mock
			t.Cleanup(func() { store.Database = originalDB })

			module.Register("existing-ready-bot", &testModuleHandler{ready: true})
			t.Cleanup(func() { module.Unregister("existing-ready-bot") })
			registerModules()

			bot, err := mock.GetBotByName(context.Background(), "existing-ready-bot")
			require.NoError(t, err)
			require.NotNil(t, bot)
			assert.Equal(t, int(schema.BotActive), bot.State)
			// Updated the existing-ready-bot AND deactivated stale-bot (both are UpdateBot calls)
			assert.GreaterOrEqual(t, mock.updateCalls, 1)
		})
	}
}

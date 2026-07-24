package web

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/cache"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type testStore struct {
	store.Adapter
	mu                        sync.Mutex
	configs                   []model.ConfigItem
	configErr                 error
	setConfigFn               func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn               func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn               func(uid types.Uid, topic, key string) error
	paramGetFn                func(ctx context.Context, flag string) (gen.Parameter, error)
	paramSetFn                func(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	paramDelFn                func(ctx context.Context, flag string) error
	agentSkills               map[string]*gen.AgentSkill
	agentSkillsErr            error
	agentSkillFiles           map[string]map[string]*gen.AgentSkillFile
	createAgentSkillFn        func(skill *gen.AgentSkill) error
	updateAgentSkillFn        func(skill *gen.AgentSkill) error
	deleteAgentSkillFn        func(flag string) error
	agentKnowledge            map[int64]*gen.AgentKnowledge
	agentKnowledgeErr         error
	agentKnowledgeSeq         int64
	agentMemoryFacts          map[string]*gen.AgentMemoryFact
	agentMemoryFactSeq        int64
	agentSessionSummaries     map[string]*gen.AgentSessionSummary
	agentSessionSummarySeq    int64
	agentSubagents            map[string]*gen.AgentSubagent
	agentSubagentsErr         error
	createAgentSubagentFn     func(subagent *gen.AgentSubagent) error
	updateAgentSubagentFn     func(subagent *gen.AgentSubagent) error
	deleteAgentSubagentFn     func(flag string) error
	agentSubagentTasks        map[int64]*gen.AgentSubagentTask
	agentSubagentTasksErr     error
	createAgentSubagentTaskFn func(task *gen.AgentSubagentTask) error
	updateAgentSubagentTaskFn func(task *gen.AgentSubagentTask) error
	chatSessions              []*gen.ChatSession
	chatSessionsByFlag        map[string]*gen.ChatSession
	chatSessionEntries        map[string][]*gen.ChatSessionEntry
	chatSessionsErr           error
	chatSessionEntriesErr     error
	chatScheduledTasks        []*gen.ChatScheduledTask
	chatScheduledTasksByFlag  map[string]*gen.ChatScheduledTask
	chatScheduledTaskRuns     map[string][]*gen.ChatScheduledTaskRun
	chatScheduledTasksErr     error
	chatScheduledTaskRunsErr  error
	agentPlans                map[string]*gen.AgentPlan
	agentPlansErr             error
	agentTodos                map[string]*gen.AgentTodo
	agentTodosErr             error
	dbClient                  *store.Client // in-memory SQLite client for view handler tests
	notifyChannels            map[int64]model.NotifyChannel
	notifyChannelErr          error
	notifyRules               map[int64]model.NotifyRule
	notifyTemplates           map[int64]model.NotifyTemplate
}

func (s *testStore) ListConfigs(_ context.Context, _ store.ListConfigOptions) ([]model.ConfigItem, error) {
	return s.configs, s.configErr
}
func (s *testStore) ConfigSet(_ context.Context, uid types.Uid, topic, key string, value types.KV) error {
	if s.setConfigFn != nil {
		return s.setConfigFn(uid, topic, key, value)
	}
	return nil
}
func (s *testStore) ConfigGet(_ context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	if s.getConfigFn != nil {
		return s.getConfigFn(uid, topic, key)
	}
	return nil, types.ErrNotFound
}
func (s *testStore) ConfigDelete(_ context.Context, uid types.Uid, topic, key string) error {
	if s.delConfigFn != nil {
		return s.delConfigFn(uid, topic, key)
	}
	return nil
}
func (s *testStore) ParameterGet(ctx context.Context, flag string) (gen.Parameter, error) {
	if s.paramGetFn != nil {
		return s.paramGetFn(ctx, flag)
	}
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    "testuser",
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

// ParameterSet stores a parameter token with the given flag, params, and expiration.
func (s *testStore) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	if s.paramSetFn != nil {
		return s.paramSetFn(ctx, flag, params, expiredAt)
	}
	return nil
}

// ParameterDelete deletes a parameter token by flag.
func (s *testStore) ParameterDelete(ctx context.Context, flag string) error {
	if s.paramDelFn != nil {
		return s.paramDelFn(ctx, flag)
	}
	return nil
}
func (s *testStore) CreateChatSession(_ context.Context, session *gen.ChatSession) error {
	if s.chatSessionsByFlag == nil {
		s.chatSessionsByFlag = map[string]*gen.ChatSession{}
	}
	row := *session
	if row.ID == 0 {
		row.ID = int64(len(s.chatSessionsByFlag) + 1)
	}
	s.chatSessionsByFlag[row.Flag] = &row
	s.chatSessions = append(s.chatSessions, &row)
	return nil
}

func (*testStore) Open(_ pkgconfig.StoreType) error { return nil }
func (*testStore) Close() error                     { return nil }
func (*testStore) IsOpen() bool                     { return false }
func (*testStore) GetName() string                  { return "test" }
func (*testStore) Stats() any                       { return nil }
func (s *testStore) GetDB() any {
	if s.dbClient != nil {
		return s.dbClient
	}
	return nil
}

// GetNotifyChannelRaw returns a channel with its raw URI for connectivity tests.
func (s *testStore) GetNotifyChannelRaw(_ context.Context, id int64) (model.NotifyChannel, error) {
	if s.notifyChannelErr != nil {
		return model.NotifyChannel{}, s.notifyChannelErr
	}
	if s.notifyChannels == nil {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	ch, ok := s.notifyChannels[id]
	if !ok {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	return ch, nil
}

// GetNotifyChannel returns a channel by ID (URI may be masked by real adapters).
func (s *testStore) GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error) {
	return s.GetNotifyChannelRaw(ctx, id)
}

// GetNotifyChannelByNameRaw returns a channel by name with its raw URI.
func (s *testStore) GetNotifyChannelByNameRaw(_ context.Context, name string) (model.NotifyChannel, error) {
	if s.notifyChannelErr != nil {
		return model.NotifyChannel{}, s.notifyChannelErr
	}
	for _, ch := range s.notifyChannels {
		if ch.Name == name {
			return ch, nil
		}
	}
	return model.NotifyChannel{}, types.ErrNotFound
}

// CreateNotifyChannel stores a new notify channel in the test map.
func (s *testStore) CreateNotifyChannel(_ context.Context, name, protocol, uri string) (int64, error) {
	if s.notifyChannels == nil {
		s.notifyChannels = map[int64]model.NotifyChannel{}
	}
	id := int64(len(s.notifyChannels) + 1)
	s.notifyChannels[id] = model.NotifyChannel{
		ID:       id,
		Name:     name,
		Protocol: protocol,
		URI:      uri,
		Enabled:  true,
	}
	return id, nil
}

// UpdateNotifyChannel updates an existing notify channel; empty uri keeps the previous value.
func (s *testStore) UpdateNotifyChannel(_ context.Context, id int64, name, protocol, uri string, enabled bool) error {
	if s.notifyChannels == nil {
		return types.ErrNotFound
	}
	ch, ok := s.notifyChannels[id]
	if !ok {
		return types.ErrNotFound
	}
	ch.Name = name
	ch.Protocol = protocol
	ch.Enabled = enabled
	if !enabled {
		ch.IsDefault = false
	}
	if uri != "" {
		ch.URI = uri
	}
	s.notifyChannels[id] = ch
	return nil
}

// GetDefaultNotifyChannelRaw returns the default enabled channel.
func (s *testStore) GetDefaultNotifyChannelRaw(_ context.Context) (model.NotifyChannel, error) {
	if s.notifyChannelErr != nil {
		return model.NotifyChannel{}, s.notifyChannelErr
	}
	for _, ch := range s.notifyChannels {
		if ch.IsDefault && ch.Enabled {
			return ch, nil
		}
	}
	return model.NotifyChannel{}, types.ErrNotFound
}

// SetDefaultNotifyChannel marks one channel as the sole default.
func (s *testStore) SetDefaultNotifyChannel(_ context.Context, id int64) error {
	if s.notifyChannels == nil {
		return types.ErrNotFound
	}
	ch, ok := s.notifyChannels[id]
	if !ok {
		return types.ErrNotFound
	}
	if !ch.Enabled {
		return types.Errorf(types.ErrInvalidArgument, "default notify channel must be enabled")
	}
	for k, existing := range s.notifyChannels {
		existing.IsDefault = k == id
		s.notifyChannels[k] = existing
	}
	return nil
}

// ListNotifyChannels returns channels from the test map.
func (s *testStore) ListNotifyChannels(_ context.Context, opts store.ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	out := make([]model.NotifyChannel, 0, len(s.notifyChannels))
	for _, ch := range s.notifyChannels {
		if opts.Protocol != "" && ch.Protocol != opts.Protocol {
			continue
		}
		if opts.Enabled != nil && ch.Enabled != *opts.Enabled {
			continue
		}
		out = append(out, ch)
	}
	return out, nil
}

// ListNotifyRules returns seeded notification rules for tests.
func (s *testStore) ListNotifyRules(_ context.Context, opts store.ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	out := make([]model.NotifyRule, 0, len(s.notifyRules))
	for _, rule := range s.notifyRules {
		if opts.Enabled != nil && rule.Enabled != *opts.Enabled {
			continue
		}
		out = append(out, rule)
	}
	return out, nil
}

// CreateNotifyRule stores a notification rule for tests.
func (s *testStore) CreateNotifyRule(_ context.Context, rule model.NotifyRule) (int64, error) {
	if s.notifyRules == nil {
		s.notifyRules = make(map[int64]model.NotifyRule)
	}
	for _, existing := range s.notifyRules {
		if existing.RuleID == rule.RuleID {
			return 0, errors.New(`postgres: create notify rule: gen: constraint failed: ERROR: duplicate key value violates unique constraint "notify_rules_rule_id_key" (SQLSTATE 23505)`)
		}
	}
	id := int64(len(s.notifyRules) + 1)
	rule.ID = id
	s.notifyRules[id] = rule
	return id, nil
}

// GetNotifyRule returns a notification rule by id.
func (s *testStore) GetNotifyRule(_ context.Context, id int64) (model.NotifyRule, error) {
	if s.notifyRules == nil {
		return model.NotifyRule{}, types.ErrNotFound
	}
	rule, ok := s.notifyRules[id]
	if !ok {
		return model.NotifyRule{}, types.ErrNotFound
	}
	return rule, nil
}

// UpdateNotifyRule updates a notification rule in the test map.
func (s *testStore) UpdateNotifyRule(_ context.Context, id int64, rule model.NotifyRule) error {
	if s.notifyRules == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyRules[id]; !ok {
		return types.ErrNotFound
	}
	rule.ID = id
	s.notifyRules[id] = rule
	return nil
}

// DeleteNotifyRule removes a notification rule from the test map.
func (s *testStore) DeleteNotifyRule(_ context.Context, id int64) error {
	if s.notifyRules == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyRules[id]; !ok {
		return types.ErrNotFound
	}
	delete(s.notifyRules, id)
	return nil
}

// DeleteNotifyChannel removes a notification channel from the test map.
func (s *testStore) DeleteNotifyChannel(_ context.Context, id int64) error {
	if s.notifyChannels == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyChannels[id]; !ok {
		return types.ErrNotFound
	}
	delete(s.notifyChannels, id)
	return nil
}

// CreateNotifyTemplate stores a notification template for tests.
func (s *testStore) CreateNotifyTemplate(_ context.Context, tmpl model.NotifyTemplate) (int64, error) {
	if s.notifyTemplates == nil {
		s.notifyTemplates = make(map[int64]model.NotifyTemplate)
	}
	id := int64(len(s.notifyTemplates) + 1)
	tmpl.ID = id
	s.notifyTemplates[id] = tmpl
	return id, nil
}

// GetNotifyTemplate returns a notification template by id.
func (s *testStore) GetNotifyTemplate(_ context.Context, id int64) (model.NotifyTemplate, error) {
	if s.notifyTemplates == nil {
		return model.NotifyTemplate{}, types.ErrNotFound
	}
	tmpl, ok := s.notifyTemplates[id]
	if !ok {
		return model.NotifyTemplate{}, types.ErrNotFound
	}
	return tmpl, nil
}

// GetNotifyTemplateByTemplateID returns a template by its template_id string.
func (s *testStore) GetNotifyTemplateByTemplateID(_ context.Context, templateID string) (model.NotifyTemplate, error) {
	for _, tmpl := range s.notifyTemplates {
		if tmpl.TemplateID == templateID {
			return tmpl, nil
		}
	}
	return model.NotifyTemplate{}, types.ErrNotFound
}

// GetDefaultNotifyTemplate returns the global default template.
func (s *testStore) GetDefaultNotifyTemplate(_ context.Context) (model.NotifyTemplate, error) {
	for _, tmpl := range s.notifyTemplates {
		if tmpl.IsDefault {
			return tmpl, nil
		}
	}
	return model.NotifyTemplate{}, types.ErrNotFound
}

// SetDefaultNotifyTemplate marks one template as the sole default.
func (s *testStore) SetDefaultNotifyTemplate(_ context.Context, id int64) error {
	if s.notifyTemplates == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyTemplates[id]; !ok {
		return types.ErrNotFound
	}
	for k, existing := range s.notifyTemplates {
		existing.IsDefault = k == id
		s.notifyTemplates[k] = existing
	}
	return nil
}

// ListNotifyTemplates returns seeded notification templates for tests.
func (s *testStore) ListNotifyTemplates(_ context.Context, _ store.ListNotifyTemplateOptions) ([]model.NotifyTemplate, error) {
	out := make([]model.NotifyTemplate, 0, len(s.notifyTemplates))
	for _, tmpl := range s.notifyTemplates {
		out = append(out, tmpl)
	}
	return out, nil
}

// UpdateNotifyTemplate updates a notification template in the test map.
func (s *testStore) UpdateNotifyTemplate(_ context.Context, id int64, tmpl model.NotifyTemplate) error {
	if s.notifyTemplates == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyTemplates[id]; !ok {
		return types.ErrNotFound
	}
	tmpl.ID = id
	s.notifyTemplates[id] = tmpl
	return nil
}

// DeleteNotifyTemplate removes a notification template from the test map.
func (s *testStore) DeleteNotifyTemplate(_ context.Context, id int64) error {
	if s.notifyTemplates == nil {
		return types.ErrNotFound
	}
	if _, ok := s.notifyTemplates[id]; !ok {
		return types.ErrNotFound
	}
	delete(s.notifyTemplates, id)
	return nil
}

func ensureChatAgentServiceForTest() {
	ensureChatAgentService()
}

func setupTestApp() (*fiber.App, *testStore) {
	ensureChatAgentServiceForTest()
	ts := &testStore{}
	// Drain async session-summary jobs before swapping the global adapter.
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = ts
	// Intentionally bypasses validateAuthConfig (Init); weak creds are for login-path unit tests only.
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	loginLimiter = nil
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts
}

// setupTestAppWithRateLimiter creates a Fiber test app with an active login rate limiter.
func setupTestAppWithRateLimiter() (*fiber.App, *testStore, *mockRateLimitStore) {
	ensureChatAgentServiceForTest()
	ts := &testStore{}
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	mockStore := newMockRateLimitStore()
	loginLimiter = newLoginRateLimiter(mockStore, 5, 10, cache.TTL(15*time.Minute), cache.TTL(15*time.Minute))
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, mockStore
}

// setupTestAppWithDB creates a Fiber test app wired with an in-memory SQLite
// database for tests that need real PageDataStore operations (view handlers).
// Each call opens a private in-memory database identified by t.Name().
func setupTestAppWithDB(t *testing.T) (*fiber.App, *testStore, *store.Client) {
	t.Helper()
	ensureChatAgentServiceForTest()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dbClient := sqlitetest.OpenClient(t, dbName)

	ts := &testStore{dbClient: dbClient}
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, dbClient
}

func createTestConfig(uid, topic, key string) model.ConfigItem {
	return model.ConfigItem{ID: 1, UID: uid, Topic: topic, Key: key, Value: types.KV{"v": "test"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// setupTestAppForRelations creates a Fiber test app with in-memory SQLite
// and pre-seeded resource links for relations tests.
func setupTestAppForRelations(t *testing.T, seedFn func(context.Context, *store.Client) error) (*fiber.App, *testStore, *store.Client) {
	t.Helper()
	app, ts, client := setupTestAppWithDB(t)
	if seedFn != nil {
		if err := seedFn(context.Background(), client); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}
	return app, ts, client
}

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
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/webauth"
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
		ID:        1,
		Flag:      flag,
		Params:    testFullWebSessionParams("testuser"),
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

func (s *testStore) GetClient() *gen.Client {
	return s.dbClient
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

var webTestGlobalsMu sync.Mutex

func lockWebTestGlobals(t *testing.T) {
	t.Helper()
	waitHealthzRefresh()
	webTestGlobalsMu.Lock()
	t.Cleanup(func() {
		chatagent.WaitForSessionSummaryGenerationForTest()
		waitHealthzRefresh()
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		loginLimiter = nil
		totpLimiter = nil
		setWebEncryptor(nil)
		webTestGlobalsMu.Unlock()
	})
}

func setupTestApp(t *testing.T) (*fiber.App, *testStore) {
	t.Helper()
	lockWebTestGlobals(t)
	ensureChatAgentServiceForTest()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dbClient := sqlitetest.OpenClient(t, dbName)
	seedWebAuthToken(t, dbClient)
	seedTestAccessToken(t, dbClient, "test-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
	seedTestAccessToken(t, dbClient, "valid-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
	testDB := &testStore{dbClient: dbClient}
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = testDB
	secure := false
	handler = moduleHandler{
		initialized: true,
		authConfig:  AuthConfig{CookieSecure: &secure},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{CookieSecure: &secure},
	}
	loginLimiter = nil
	totpLimiter = nil
	enc, _, _, _ := webauth.LoadEncryptor("test-encryption-key-for-unit-tests", ".")
	setWebEncryptor(enc)
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, testDB
}

// setupTestAppWithRateLimiter creates a Fiber test app with an active login rate limiter.
func setupTestAppWithRateLimiter(t *testing.T) (*fiber.App, *testStore, *mockRateLimitStore) {
	t.Helper()
	lockWebTestGlobals(t)
	ensureChatAgentServiceForTest()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dbClient := sqlitetest.OpenClient(t, dbName)
	seedWebAuthToken(t, dbClient)
	testDB := &testStore{dbClient: dbClient}
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = testDB
	secure := false
	handler = moduleHandler{
		initialized: true,
		authConfig:  AuthConfig{CookieSecure: &secure},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{CookieSecure: &secure},
	}
	mockStore := newMockRateLimitStore()
	loginLimiter = newLoginRateLimiter(mockStore, 5, 10, cache.TTL(15*time.Minute), cache.TTL(15*time.Minute))
	totpLimiter = newLoginRateLimiter(mockStore, 3, 10, cache.TTL(15*time.Minute), cache.TTL(15*time.Minute))
	enc, _, _, _ := webauth.LoadEncryptor("test-encryption-key-for-unit-tests", ".")
	setWebEncryptor(enc)
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, testDB, mockStore
}

// setupTestAppWithDB creates a Fiber test app wired with an in-memory SQLite
// database for tests that need real PageDataStore / web account operations.
func setupTestAppWithDB(t *testing.T) (*fiber.App, *testStore, *store.Client) {
	t.Helper()
	lockWebTestGlobals(t)
	ensureChatAgentServiceForTest()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dbClient := sqlitetest.OpenClient(t, dbName)

	ts := &testStore{dbClient: dbClient}
	seedWebAuthToken(t, dbClient)
	seedTestAccessToken(t, dbClient, "test-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
	seedTestAccessToken(t, dbClient, "valid-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
	chatagent.WaitForSessionSummaryGenerationForTest()
	store.Database = ts
	secure := false
	handler = moduleHandler{
		initialized: true,
		authConfig:  AuthConfig{CookieSecure: &secure},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{CookieSecure: &secure},
	}
	loginLimiter = nil
	totpLimiter = nil
	enc, _, _, err := webauth.LoadEncryptor("test-encryption-key-for-unit-tests", t.TempDir())
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	setWebEncryptor(enc)
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, dbClient
}

func seedWebAccount(t *testing.T, client *store.Client, username, password string, totpEnabled bool) {
	t.Helper()
	hash, err := webauth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	ws := store.NewWebAccountStore(client)
	_, err = ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     username,
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if !totpEnabled {
		return
	}
	enc := getEncryptor()
	secret, err := webauth.GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("totp secret: %v", err)
	}
	ct, nonce, err := enc.Encrypt([]byte(secret))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, hashes, err := enc.GenerateBackupCodes(2)
	if err != nil {
		t.Fatalf("backup codes: %v", err)
	}
	if err := ws.EnableTOTP(context.Background(), username, ct, nonce, hashes, 0); err != nil {
		t.Fatalf("enable totp: %v", err)
	}
}

func testFullWebSessionParams(uid string) map[string]any {
	return map[string]any{
		"uid":    uid,
		"topic":  "web",
		"kind":   webauth.KindFull,
		"scopes": []string{"admin:*"},
	}
}

func seedWebAuthToken(t *testing.T, client *store.Client) {
	t.Helper()
	seedTestAccessToken(t, client, "valid-test-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
}

func seedTestAccessToken(t *testing.T, client *store.Client, rawToken string, params types.KV, expiredAt time.Time) {
	t.Helper()
	if err := store.NewModuleDataStore(client).ParameterSet(
		context.Background(),
		auth.HashToken(rawToken),
		params,
		expiredAt,
	); err != nil {
		t.Fatalf("seed access token %q: %v", rawToken, err)
	}
}

func seedLegacyPlaintextAccessToken(t *testing.T, client *store.Client, rawToken string, params types.KV, expiredAt time.Time) {
	t.Helper()
	if err := store.NewModuleDataStore(client).ParameterSet(
		context.Background(),
		rawToken,
		params,
		expiredAt,
	); err != nil {
		t.Fatalf("seed legacy access token %q: %v", rawToken, err)
	}
}

// ensureTestStoreDB attaches an in-memory SQLite client when tests use testStore
// without dbClient but handlers call XxxStoreFromDB().
func ensureTestStoreDB(t *testing.T, ts *testStore) {
	t.Helper()
	if ts == nil || ts.dbClient != nil {
		return
	}
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	ts.dbClient = sqlitetest.OpenClient(t, dbName)
	seedWebAuthToken(t, ts.dbClient)
	seedTestAccessToken(t, ts.dbClient, "test-token", testFullWebSessionParams("testuser"), time.Now().Add(time.Hour))
}

func collectTestChatSessions(ts *testStore) []*gen.ChatSession {
	if ts == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]*gen.ChatSession, 0)
	for _, sess := range ts.chatSessions {
		if sess == nil {
			continue
		}
		if _, ok := seen[sess.Flag]; ok {
			continue
		}
		seen[sess.Flag] = struct{}{}
		out = append(out, sess)
	}
	for _, sess := range ts.chatSessionsByFlag {
		if sess == nil {
			continue
		}
		if _, ok := seen[sess.Flag]; ok {
			continue
		}
		seen[sess.Flag] = struct{}{}
		out = append(out, sess)
	}
	return out
}

func collectTestChatScheduledTasks(ts *testStore) []*gen.ChatScheduledTask {
	if ts == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]*gen.ChatScheduledTask, 0)
	for _, task := range ts.chatScheduledTasks {
		if task == nil {
			continue
		}
		if _, ok := seen[task.Flag]; ok {
			continue
		}
		seen[task.Flag] = struct{}{}
		out = append(out, task)
	}
	for _, task := range ts.chatScheduledTasksByFlag {
		if task == nil {
			continue
		}
		if _, ok := seen[task.Flag]; ok {
			continue
		}
		seen[task.Flag] = struct{}{}
		out = append(out, task)
	}
	return out
}

func syncTestChatSession(t *testing.T, client *gen.Client, sess *gen.ChatSession) {
	t.Helper()
	if client == nil || sess == nil {
		return
	}
	row := *sess
	if row.UID == "" {
		row.UID = "testuser"
	}
	builder := client.ChatSession.Create().
		SetFlag(row.Flag).
		SetUID(row.UID).
		SetLeafID(row.LeafID).
		SetState(row.State).
		SetMode(row.Mode).
		SetModel(row.Model).
		SetThinkingLevel(row.ThinkingLevel).
		SetTitle(row.Title).
		SetPreview(row.Preview).
		SetPinned(row.Pinned).
		SetArchived(row.Archived)
	if row.ID != 0 {
		builder = builder.SetID(row.ID)
	}
	if !row.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(row.CreatedAt)
	}
	if !row.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(row.UpdatedAt)
	}
	if _, err := builder.Save(context.Background()); err != nil {
		t.Fatalf("sync chat session %q: %v", row.Flag, err)
	}
}

func syncTestChatScheduledTask(ctx context.Context, t *testing.T, task *gen.ChatScheduledTask) {
	t.Helper()
	row := *task
	if row.UID == "" {
		row.UID = "testuser"
	}
	if row.ScheduleKind == "" {
		row.ScheduleKind = "cron"
	}
	if row.Prompt == "" {
		row.Prompt = "test prompt"
	}
	if row.Name == "" {
		row.Name = row.Flag
	}
	if err := store.ChatStoreFromDB().CreateChatScheduledTask(ctx, &row); err != nil {
		t.Fatalf("sync chat scheduled task %q: %v", row.Flag, err)
	}
}

// syncTestStoreToDB copies seeded in-memory testStore fixtures into SQLite so
// handlers that call XxxStoreFromDB() see the same data as legacy mock methods.
func syncTestStoreToDB(t *testing.T, ts *testStore) {
	t.Helper()
	if ts == nil {
		return
	}
	ensureTestStoreDB(t, ts)
	if ts.dbClient == nil {
		return
	}
	ctx := context.Background()
	syncTestStoreConfigs(ctx, t, ts)
	syncTestStoreAgentKnowledge(ctx, t, ts)
	syncTestStoreAgentSkills(ctx, t, ts)
	syncTestStoreAgentMemory(ctx, t, ts)
	syncTestStoreAgentSummaries(ctx, t, ts)
	syncTestStoreChat(ctx, t, ts)
	syncTestStoreAgentTodos(ctx, t, ts)
	syncTestStoreAgentSubagents(ctx, t, ts)
	syncTestStoreAgentPlans(ctx, t, ts)
	syncTestStoreNotify(ctx, t, ts)
}

func syncTestStoreConfigs(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, item := range ts.configs {
		if err := store.ModuleDataStoreFromDB().ConfigSet(ctx, types.Uid(item.UID), item.Topic, item.Key, item.Value); err != nil {
			t.Fatalf("sync config %s/%s/%s: %v", item.UID, item.Topic, item.Key, err)
		}
	}
}

func syncTestStoreAgentKnowledge(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, doc := range ts.agentKnowledge {
		row := *doc
		if row.Content == "" {
			row.Content = " "
		}
		if row.Title == "" {
			row.Title = " "
		}
		if err := store.AgentStoreFromDB().CreateAgentKnowledge(ctx, &row); err != nil {
			t.Fatalf("sync agent knowledge %q: %v", row.Path, err)
		}
	}
}

func syncTestStoreAgentSkills(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, skill := range ts.agentSkills {
		row := *skill
		if row.Content == "" {
			row.Content = " "
		}
		if row.Name == "" {
			row.Name = row.Flag
		}
		if row.Description == "" {
			row.Description = " "
		}
		if err := store.AgentStoreFromDB().CreateAgentSkill(ctx, &row); err != nil {
			t.Fatalf("sync agent skill %q: %v", row.Flag, err)
		}
	}
	for skillFlag, files := range ts.agentSkillFiles {
		for _, file := range files {
			row := *file
			row.SkillFlag = skillFlag
			if err := store.AgentStoreFromDB().CreateAgentSkillFile(ctx, &row); err != nil {
				t.Fatalf("sync agent skill file %q: %v", row.Path, err)
			}
		}
	}
}

func syncTestStoreAgentMemory(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, fact := range ts.agentMemoryFacts {
		row := *fact
		if _, err := store.AgentStoreFromDB().UpsertAgentMemoryFact(ctx, store.AgentMemoryFactUpsert{
			Scope:  row.Scope,
			Key:    row.Key,
			Value:  row.Value,
			Pinned: row.Pinned,
		}); err != nil {
			t.Fatalf("sync agent memory fact %s/%s: %v", row.Scope, row.Key, err)
		}
	}
}

func syncTestStoreAgentSummaries(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, summary := range ts.agentSessionSummaries {
		row := *summary
		if row.Scope == "" {
			row.Scope = "default"
		}
		builder := ts.dbClient.AgentSessionSummary.Create().
			SetSessionFlag(row.SessionFlag).
			SetScope(row.Scope).
			SetTitle(row.Title).
			SetSummary(row.Summary).
			SetStatus(string(row.Status)).
			SetError(row.Error).
			SetClaimToken(row.ClaimToken)
		if row.ID != 0 {
			builder = builder.SetID(row.ID)
		}
		if row.ClaimedAt != nil {
			builder = builder.SetClaimedAt(*row.ClaimedAt)
		}
		if !row.CreatedAt.IsZero() {
			builder = builder.SetCreatedAt(row.CreatedAt)
		}
		if !row.UpdatedAt.IsZero() {
			builder = builder.SetUpdatedAt(row.UpdatedAt)
		}
		if _, err := builder.Save(ctx); err != nil {
			t.Fatalf("sync agent session summary %q: %v", row.SessionFlag, err)
		}
	}
}

func syncTestStoreChat(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, sess := range collectTestChatSessions(ts) {
		syncTestChatSession(t, ts.dbClient, sess)
	}
	for sessionID, entries := range ts.chatSessionEntries {
		for _, entry := range entries {
			row := *entry
			row.SessionID = sessionID
			if err := store.ChatStoreFromDB().CreateChatSessionEntry(ctx, &row); err != nil {
				t.Fatalf("sync chat session entry %q: %v", row.Flag, err)
			}
		}
	}
	for _, task := range collectTestChatScheduledTasks(ts) {
		syncTestChatScheduledTask(ctx, t, task)
	}
	for taskID, runs := range ts.chatScheduledTaskRuns {
		for _, run := range runs {
			row := *run
			row.TaskID = taskID
			if row.Flag == "" {
				row.Flag = "run-" + taskID
			}
			if err := store.ChatStoreFromDB().CreateChatScheduledTaskRun(ctx, &row); err != nil {
				t.Fatalf("sync chat scheduled task run: %v", err)
			}
		}
	}
}

func syncTestStoreAgentTodos(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	todosBySession := map[string][]*gen.AgentTodo{}
	for _, todo := range ts.agentTodos {
		row := *todo
		if row.SessionID == "" {
			continue
		}
		if row.Content == "" {
			row.Content = "todo"
		}
		if row.ItemID == "" {
			row.ItemID = row.Flag
		}
		if row.Status == "" {
			row.Status = "pending"
		}
		todosBySession[row.SessionID] = append(todosBySession[row.SessionID], &row)
	}
	for sessionID, todos := range todosBySession {
		if err := store.AgentStoreFromDB().ReplaceAgentTodosForSession(ctx, sessionID, todos); err != nil {
			t.Fatalf("sync agent todos for %q: %v", sessionID, err)
		}
	}
}

func syncTestStoreAgentSubagents(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, subagent := range ts.agentSubagents {
		row := *subagent
		if row.Name == "" {
			row.Name = row.Flag
		}
		if row.Description == "" {
			row.Description = " "
		}
		if row.SystemPrompt == "" {
			row.SystemPrompt = " "
		}
		if err := store.AgentStoreFromDB().CreateAgentSubagent(ctx, &row); err != nil {
			t.Fatalf("sync agent subagent %q: %v", row.Flag, err)
		}
	}
	for _, task := range ts.agentSubagentTasks {
		row := *task
		if row.SessionID == "" {
			row.SessionID = "sess-test"
		}
		if row.Prompt == "" {
			row.Prompt = "test prompt"
		}
		if row.SubagentName == "" {
			row.SubagentName = "test-subagent"
		}
		if row.Status == "" {
			row.Status = string(schema.AgentSubagentTaskStatusRunning)
		}
		if err := store.AgentStoreFromDB().CreateAgentSubagentTask(ctx, &row); err != nil {
			t.Fatalf("sync agent subagent task: %v", err)
		}
	}
}

func syncTestStoreAgentPlans(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	for _, plan := range ts.agentPlans {
		row := *plan
		if row.Content == "" {
			row.Content = " "
		}
		if row.Title == "" {
			row.Title = " "
		}
		if err := store.AgentStoreFromDB().CreateAgentPlan(ctx, &row); err != nil {
			t.Fatalf("sync agent plan %q: %v", row.Flag, err)
		}
	}
}

func syncTestStoreNotify(ctx context.Context, t *testing.T, ts *testStore) {
	t.Helper()
	defaultNotifyChannelID := int64(0)
	for _, ch := range ts.notifyChannels {
		id, err := store.NotifyConfigStoreFromDB().CreateNotifyChannel(ctx, ch.Name, ch.Protocol, ch.URI)
		if err != nil {
			t.Fatalf("sync notify channel %q: %v", ch.Name, err)
		}
		if !ch.Enabled {
			if err := store.NotifyConfigStoreFromDB().UpdateNotifyChannel(ctx, id, ch.Name, ch.Protocol, ch.URI, false); err != nil {
				t.Fatalf("sync notify channel disabled %q: %v", ch.Name, err)
			}
		}
		if ch.IsDefault {
			defaultNotifyChannelID = id
		}
	}
	if defaultNotifyChannelID != 0 {
		if err := store.NotifyConfigStoreFromDB().SetDefaultNotifyChannel(ctx, defaultNotifyChannelID); err != nil {
			t.Fatalf("sync default notify channel: %v", err)
		}
	}
	for _, rule := range ts.notifyRules {
		if _, err := store.NotifyConfigStoreFromDB().CreateNotifyRule(ctx, rule); err != nil {
			t.Fatalf("sync notify rule %q: %v", rule.RuleID, err)
		}
	}
	defaultNotifyTemplateID := int64(0)
	for _, tmpl := range ts.notifyTemplates {
		id, err := store.NotifyConfigStoreFromDB().CreateNotifyTemplate(ctx, tmpl)
		if err != nil {
			t.Fatalf("sync notify template %q: %v", tmpl.TemplateID, err)
		}
		if tmpl.IsDefault {
			defaultNotifyTemplateID = id
		}
	}
	if defaultNotifyTemplateID != 0 {
		if err := store.NotifyConfigStoreFromDB().SetDefaultNotifyTemplate(ctx, defaultNotifyTemplateID); err != nil {
			t.Fatalf("sync default notify template: %v", err)
		}
	}
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

func listTestNotifyChannels(t *testing.T) []model.NotifyChannel {
	t.Helper()
	chs, err := store.NotifyConfigStoreFromDB().ListNotifyChannels(context.Background(), store.ListNotifyChannelOptions{})
	if err != nil {
		t.Fatalf("list notify channels: %v", err)
	}
	return chs
}

func testNotifyChannelRaw(t *testing.T, id int64) model.NotifyChannel {
	t.Helper()
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannelRaw(context.Background(), id)
	if err != nil {
		t.Fatalf("get notify channel raw %d: %v", id, err)
	}
	return ch
}

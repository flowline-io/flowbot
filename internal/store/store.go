// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/app"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventconsumption"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/llmusagerecord"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/clip"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pagedata"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinitionversion"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinesteprun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pollingstate"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/predicate"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowsteprun"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// Client is a type alias for the Ent client.
type Client = gen.Client

var adp Adapter

var availableAdapters = make(map[string]Adapter)

func openAdapter(jsonConfig config.StoreType) error {
	if adp == nil {
		if len(availableAdapters) >= 1 {
			// Default to the only entry in availableAdapters.
			for _, v := range availableAdapters {
				adp = v
			}
		} else {
			return errors.New("store: db adapter is not specified. Please set postgres.dsn in flowbot.yaml")
		}
	}

	if adp.IsOpen() {
		return errors.New("store: connection is already opened")
	}

	return adp.Open(jsonConfig)
}

func RegisterAdapter(a Adapter) {
	if a == nil {
		flog.Fatal("store: Register adapter is nil")
	}

	name := a.GetName()
	if _, ok := availableAdapters[name]; ok {
		flog.Fatal("store: adapter %s is already registered", name)
	}
	availableAdapters[name] = a
	flog.Info("store: adapter '%s' registered", name)
}

func Migrate() error {
	if !adp.IsOpen() {
		return errors.New("store: connection is not opened")
	}
	client, ok := adp.GetDB().(*gen.Client)
	if !ok {
		return errors.New("store: failed to get Ent client from adapter")
	}
	err := client.Schema.Create(context.Background())
	if err != nil {
		return fmt.Errorf("store: schema migration: %w", err)
	}
	return nil
}

// FileSystem Media handler
var FileSystem media.Handler

// Registered media/file handlers.
var fileHandlers map[string]media.Handler

// RegisterMediaHandler saves reference to a media handler (file upload-download handler).
func RegisterMediaHandler(name string, mh media.Handler) {
	if fileHandlers == nil {
		fileHandlers = make(map[string]media.Handler)
	}

	if mh == nil {
		flog.Fatal("RegisterMediaHandler: handler is nil")
	}
	if _, dup := fileHandlers[name]; dup {
		flog.Fatal("RegisterMediaHandler: called twice for handler %s", name)
	}
	fileHandlers[name] = mh
	flog.Info("media: handler '%s' registered", name)
}

// UseMediaHandler sets specified media handler as default.
func UseMediaHandler(name, mediaConfig string) error {
	mediaHandler := fileHandlers[name]
	if mediaHandler == nil {
		return fmt.Errorf("unknown handler %s", name)
	}
	FileSystem = mediaHandler
	return mediaHandler.Init(mediaConfig)
}

// PersistentStorageInterface defines methods used for interaction with persistent storage.
type PersistentStorageInterface interface {
	Open(jsonConfig config.StoreType) error
	Close() error
	IsOpen() bool
	GetAdapter() Adapter
	DbStats() func() any
}

// Store is the main object for interacting with persistent storage.
var Store PersistentStorageInterface

type storeObj struct{}

func (storeObj) Open(jsonConfig config.StoreType) error {
	return openAdapter(jsonConfig)
}

func (storeObj) Close() error {
	if adp.IsOpen() {
		return adp.Close()
	}

	return nil
}

func (storeObj) GetAdapter() Adapter {
	return adp
}

// IsOpen checks if persistent storage connection has been initialized.
func (storeObj) IsOpen() bool {
	if adp != nil {
		return adp.IsOpen()
	}

	return false
}

func (s storeObj) DbStats() func() any {
	if !s.IsOpen() {
		return nil
	}
	return adp.Stats
}

// ListConfigOptions controls pagination and search for ListConfigs.
type ListConfigOptions struct {
	Offset int
	Limit  int
	Search string
}

// ListNotifyChannelOptions holds filtering options for listing notification channels.
type ListNotifyChannelOptions struct {
	Protocol string
	Enabled  *bool // nil = all, true = enabled only, false = disabled only
}

// ListNotifyRuleOptions holds filtering and sorting options for listing notification rules.
type ListNotifyRuleOptions struct {
	Enabled *bool // nil = all, true = enabled only, false = disabled only
}

// ListNotifyTemplateOptions holds filtering options for listing notification templates.
type ListNotifyTemplateOptions struct{}

// ListChatSessionsOptions holds pagination for listing chat agent sessions.
type ListChatSessionsOptions struct {
	Limit  int    // max 100, default 20
	Cursor string // opaque cursor: session ID value as string
	UID    string // when set, only sessions owned by this user are returned
	State  *int   // when set, only sessions in this state are returned
}

// ListChatScheduledTasksOptions filters scheduled task queries.
type ListChatScheduledTasksOptions struct {
	UID    string
	States []string
}

// UpdateChatScheduledTaskParams carries partial updates for a scheduled task row.
type UpdateChatScheduledTaskParams struct {
	Name      *string
	Cron      *string
	RunAt     *time.Time
	Prompt    *string
	State     *string
	LastRunAt *time.Time
	NextRunAt *time.Time
}

// UpdateChatScheduledTaskRunParams carries partial updates for one run row.
type UpdateChatScheduledTaskRunParams struct {
	State      *string
	Reply      *string
	Error      *string
	FinishedAt *time.Time
}

type Adapter interface {
	// General

	// Open and configure the adapter
	Open(storeConfig config.StoreType) error
	// Close the adapter
	Close() error
	// IsOpen checks if the adapter is ready for use
	IsOpen() bool
	// GetName returns the name of the adapter
	GetName() string
	// Stats returns the DB connection stats object.
	Stats() any
	// Ping checks database connectivity and returns the round-trip latency.
	Ping(ctx context.Context) (time.Duration, error)
	// GetDB returns the underlying DB connection
	GetDB() any

	// User management

	// UserCreate creates user record
	UserCreate(ctx context.Context, user *gen.User) error
	// UserGet returns record for a given user ID
	UserGet(ctx context.Context, uid types.Uid) (*gen.User, error)
	// UserGetAll returns user records for a given list of user IDs
	UserGetAll(ctx context.Context, ids ...types.Uid) ([]*gen.User, error)
	// FirstUser returns the first user
	FirstUser(ctx context.Context) (*gen.User, error)
	// UserDelete deletes user record
	UserDelete(ctx context.Context, uid types.Uid, hard bool) error
	// UserUpdate updates user record
	UserUpdate(ctx context.Context, uid types.Uid, update types.KV) error

	// File upload records. The files are stored outside the database.

	// FileStartUpload initializes a file upload.
	FileStartUpload(ctx context.Context, fd *types.FileDef) error
	// FileFinishUpload marks file upload as completed, successfully or otherwise.
	FileFinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error)
	// FileGet fetches a record of a specific file
	FileGet(ctx context.Context, fid string) (*types.FileDef, error)
	// FileDeleteUnused deletes records where UseCount is zero. If olderThan is non-zero, deletes
	// unused records with UpdatedAt before olderThan.
	// Returns array of FileDef.Location of deleted filerecords so actual files can be deleted too.
	FileDeleteUnused(ctx context.Context, olderThan time.Time, limit int) ([]string, error)

	GetUsers(ctx context.Context) ([]*gen.User, error)
	GetUserById(ctx context.Context, id int64) (*gen.User, error)
	GetUserByFlag(ctx context.Context, flag string) (*gen.User, error)
	CreatePlatformUser(ctx context.Context, item *gen.PlatformUser) (int64, error)
	GetPlatformUsersByUserId(ctx context.Context, userId int64) ([]*gen.PlatformUser, error)
	GetPlatformUserByFlag(ctx context.Context, flag string) (*gen.PlatformUser, error)
	UpdatePlatformUser(ctx context.Context, item *gen.PlatformUser) error
	GetPlatformChannelByFlag(ctx context.Context, flag string) (*gen.PlatformChannel, error)
	GetPlatformChannelsByPlatformIds(ctx context.Context, platformIds []int64) ([]*gen.PlatformChannel, error)
	GetPlatformChannelsByChannelId(ctx context.Context, channelId int64) (*gen.PlatformChannel, error)
	CreatePlatformChannel(ctx context.Context, item *gen.PlatformChannel) (int64, error)
	UpdatePlatformChannelChannelID(ctx context.Context, platformChannelID, channelID int64) error
	CreatePlatformChannelUser(ctx context.Context, item *gen.PlatformChannelUser) (int64, error)
	GetPlatformChannelUsersByUserFlag(ctx context.Context, userFlag string) ([]*gen.PlatformChannelUser, error)
	GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error)
	GetMessage(ctx context.Context, flag string) (*gen.Message, error)
	GetMessageByPlatform(ctx context.Context, platformId int64, platformMsgId string) (*gen.Message, error)
	GetMessagesBySession(ctx context.Context, session string) ([]*gen.Message, error)
	CreateMessage(ctx context.Context, message gen.Message) error

	CreateChatSession(ctx context.Context, session *gen.ChatSession) error
	GetChatSession(ctx context.Context, flag string) (*gen.ChatSession, error)
	ListChatSessions(ctx context.Context, opts ListChatSessionsOptions) ([]*gen.ChatSession, string, error)
	UpdateChatSessionLeaf(ctx context.Context, flag, leafID string) error
	UpdateChatSessionMode(ctx context.Context, flag, mode string) error
	UpdateChatSessionSettings(ctx context.Context, flag, modelName, thinkingLevel string) error
	UpdateChatSessionTitle(ctx context.Context, flag, title string) error
	CloseChatSession(ctx context.Context, flag string) error
	CreateChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error
	AppendChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error
	ListChatSessionEntries(ctx context.Context, sessionID string) ([]*gen.ChatSessionEntry, error)
	// ListChatSessionEntriesBySessions returns entries for the given session flags ordered by created_at.
	ListChatSessionEntriesBySessions(ctx context.Context, sessionIDs []string) ([]*gen.ChatSessionEntry, error)
	GetChatSessionEntry(ctx context.Context, flag string) (*gen.ChatSessionEntry, error)
	GetChatSessionEntryInSession(ctx context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error)

	CreateChatScheduledTask(ctx context.Context, task *gen.ChatScheduledTask) error
	DeleteChatScheduledTask(ctx context.Context, flag string) error
	GetChatScheduledTask(ctx context.Context, flag string) (*gen.ChatScheduledTask, error)
	GetChatScheduledTaskForUID(ctx context.Context, flag, uid string) (*gen.ChatScheduledTask, error)
	ListChatScheduledTasks(ctx context.Context, opts ListChatScheduledTasksOptions) ([]*gen.ChatScheduledTask, error)
	UpdateChatScheduledTask(ctx context.Context, flag string, params UpdateChatScheduledTaskParams) error
	CreateChatScheduledTaskRun(ctx context.Context, run *gen.ChatScheduledTaskRun) error
	UpdateChatScheduledTaskRun(ctx context.Context, flag string, params UpdateChatScheduledTaskRunParams) error
	FailStaleChatScheduledTaskRuns(ctx context.Context) error
	ListChatScheduledTaskRuns(ctx context.Context, taskID string, limit int) ([]*gen.ChatScheduledTaskRun, error)

	CreateAgentPlan(ctx context.Context, plan *gen.AgentPlan) error
	GetAgentPlan(ctx context.Context, flag string) (*gen.AgentPlan, error)
	GetAgentPlanInSession(ctx context.Context, sessionID, flag string) (*gen.AgentPlan, error)
	ListAgentPlansBySession(ctx context.Context, sessionID string) ([]*gen.AgentPlan, error)

	ListAgentTodosBySession(ctx context.Context, sessionID string) ([]*gen.AgentTodo, error)
	ListAgentTodosBySessions(ctx context.Context, sessionIDs []string) ([]*gen.AgentTodo, error)
	ReplaceAgentTodosForSession(ctx context.Context, sessionID string, items []*gen.AgentTodo) error
	MergeAgentTodosForSession(ctx context.Context, sessionID string, items []*gen.AgentTodo) error

	ListAgentSkills(ctx context.Context, enabledOnly bool) ([]*gen.AgentSkill, error)
	GetAgentSkillsMaxUpdatedAt(ctx context.Context) (time.Time, error)
	GetAgentSkillByName(ctx context.Context, name string) (*gen.AgentSkill, error)
	GetAgentSkillByFlag(ctx context.Context, flag string) (*gen.AgentSkill, error)
	CreateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error
	UpdateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error
	DeleteAgentSkill(ctx context.Context, flag string) error

	ListAgentSkillFiles(ctx context.Context, skillFlag string) ([]*gen.AgentSkillFile, error)
	GetAgentSkillFile(ctx context.Context, skillFlag, path string) (*gen.AgentSkillFile, error)
	CreateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error
	UpdateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error
	DeleteAgentSkillFile(ctx context.Context, skillFlag, path string) error
	DeleteAgentSkillFilesByFlag(ctx context.Context, skillFlag string) error

	ListAgentSubagents(ctx context.Context, enabledOnly bool) ([]*gen.AgentSubagent, error)
	GetAgentSubagentsMaxUpdatedAt(ctx context.Context) (time.Time, error)
	GetAgentSubagentByName(ctx context.Context, name string) (*gen.AgentSubagent, error)
	GetAgentSubagentByFlag(ctx context.Context, flag string) (*gen.AgentSubagent, error)
	CreateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error
	UpdateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error
	DeleteAgentSubagent(ctx context.Context, flag string) error

	CreateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error
	UpdateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error
	ListAgentSubagentTasks(ctx context.Context, sessionID string, limit int) ([]*gen.AgentSubagentTask, error)
	GetAgentSubagentTask(ctx context.Context, id int64) (*gen.AgentSubagentTask, error)

	GetBot(ctx context.Context, id int64) (*gen.Bot, error)
	GetBotByName(ctx context.Context, name string) (*gen.Bot, error)
	CreateBot(ctx context.Context, bot *gen.Bot) (int64, error)
	UpdateBot(ctx context.Context, bot *gen.Bot) error
	DeleteBot(ctx context.Context, name string) error
	GetBots(ctx context.Context) ([]*gen.Bot, error)
	GetPlatform(ctx context.Context, id int64) (*gen.Platform, error)
	GetPlatformByName(ctx context.Context, name string) (*gen.Platform, error)
	GetPlatforms(ctx context.Context) ([]*gen.Platform, error)
	CreatePlatform(ctx context.Context, platform *gen.Platform) (int64, error)
	GetChannel(ctx context.Context, id int64) (*gen.Channel, error)
	GetChannelByName(ctx context.Context, name string) (*gen.Channel, error)
	CreateChannel(ctx context.Context, channel *gen.Channel) (int64, error)
	UpdateChannel(ctx context.Context, channel *gen.Channel) error
	DeleteChannel(ctx context.Context, name string) error
	GetChannels(ctx context.Context) ([]*gen.Channel, error)

	DataSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error
	DataGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
	DataList(ctx context.Context, uid types.Uid, topic string, filter types.DataFilter) ([]*gen.Data, error)
	DataDelete(ctx context.Context, uid types.Uid, topic, key string) error
	ConfigSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error
	ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
	ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]*gen.ConfigData, error)
	ConfigDelete(ctx context.Context, uid types.Uid, topic, key string) error
	// ListConfigs returns config items across all uids/topics with optional search and pagination.
	ListConfigs(ctx context.Context, opts ListConfigOptions) ([]model.ConfigItem, error)
	OAuthSet(ctx context.Context, oauth gen.OAuth) error
	// OAuthGet returns the raw oauth record. Most callers should use
	// providers.GetOrRefreshToken() which handles expired token refresh.
	OAuthGet(ctx context.Context, uid types.Uid, topic, t string) (gen.OAuth, error)
	OAuthGetAvailable(ctx context.Context, t string) ([]gen.OAuth, error)
	FormSet(ctx context.Context, formId string, form gen.Form) error
	FormGet(ctx context.Context, formId string) (gen.Form, error)
	PageSet(ctx context.Context, pageId string, page gen.Page) error
	PageGet(ctx context.Context, pageId string) (gen.Page, error)
	BehaviorSet(ctx context.Context, behavior gen.Behavior) error
	BehaviorGet(ctx context.Context, uid types.Uid, flag string) (gen.Behavior, error)
	BehaviorList(ctx context.Context, uid types.Uid) ([]*gen.Behavior, error)
	BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error
	ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	ParameterGet(ctx context.Context, flag string) (gen.Parameter, error)
	ParameterDelete(ctx context.Context, flag string) error
	// ListTokens returns all token parameters (flag LIKE 'fb_%'), sorted by created_at desc.
	ListTokens(ctx context.Context) ([]model.TokenItem, error)
	// CreateToken generates a new token and persists it as a parameter row.
	// Returns the plaintext token string.
	CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error)
	// RevokeToken deletes the parameter row identified by the token flag.
	RevokeToken(ctx context.Context, flag string) error
	CreateInstruct(ctx context.Context, instruct *gen.Instruct) (int64, error)
	ListInstruct(ctx context.Context, uid types.Uid, isExpire bool, limit int) ([]*gen.Instruct, error)
	UpdateInstruct(ctx context.Context, instruct *gen.Instruct) error
	CreateCounter(ctx context.Context, counter *gen.Counter) (int64, error)
	IncreaseCounter(ctx context.Context, id, amount int64) error
	DecreaseCounter(ctx context.Context, id, amount int64) error
	ListCounter(ctx context.Context, uid types.Uid, topic string) ([]*gen.Counter, error)
	GetCounter(ctx context.Context, id int64) (gen.Counter, error)
	GetCounterByFlag(ctx context.Context, uid types.Uid, topic, flag string) (gen.Counter, error)

	GetAgents(ctx context.Context) ([]*gen.Agent, error)
	GetAgentByHostid(ctx context.Context, uid types.Uid, topic, hostid string) (*gen.Agent, error)
	CreateAgent(ctx context.Context, agent *gen.Agent) (int64, error)
	UpdateAgentLastOnlineAt(ctx context.Context, uid types.Uid, topic, hostid string, lastOnlineAt time.Time) error
	UpdateAgentOnlineDuration(ctx context.Context, uid types.Uid, topic, hostid string, offlineTime time.Time) error

	// NotifyChannel CRUD
	CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error)
	GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error)    // returns masked URI
	GetNotifyChannelRaw(ctx context.Context, id int64) (model.NotifyChannel, error) // returns raw URI (internal use only)
	// GetNotifyChannelByNameRaw returns a channel by unique name with raw URI (internal send path).
	GetNotifyChannelByNameRaw(ctx context.Context, name string) (model.NotifyChannel, error)
	// GetDefaultNotifyChannelRaw returns the global default enabled channel with raw URI.
	GetDefaultNotifyChannelRaw(ctx context.Context) (model.NotifyChannel, error)
	ListNotifyChannels(ctx context.Context, opts ListNotifyChannelOptions) ([]model.NotifyChannel, error)
	UpdateNotifyChannel(ctx context.Context, id int64, name, protocol, uri string, enabled bool) error
	// SetDefaultNotifyChannel marks id as the sole global default channel.
	SetDefaultNotifyChannel(ctx context.Context, id int64) error
	DeleteNotifyChannel(ctx context.Context, id int64) error

	// NotifyRule CRUD
	CreateNotifyRule(ctx context.Context, rule model.NotifyRule) (int64, error)
	GetNotifyRule(ctx context.Context, id int64) (model.NotifyRule, error)
	ListNotifyRules(ctx context.Context, opts ListNotifyRuleOptions) ([]model.NotifyRule, error)
	UpdateNotifyRule(ctx context.Context, id int64, rule model.NotifyRule) error
	DeleteNotifyRule(ctx context.Context, id int64) error

	// NotifyTemplate CRUD
	CreateNotifyTemplate(ctx context.Context, tmpl model.NotifyTemplate) (int64, error)
	GetNotifyTemplate(ctx context.Context, id int64) (model.NotifyTemplate, error)
	// GetNotifyTemplateByTemplateID returns a template by its stable template_id string.
	GetNotifyTemplateByTemplateID(ctx context.Context, templateID string) (model.NotifyTemplate, error)
	// GetDefaultNotifyTemplate returns the global default notification template.
	GetDefaultNotifyTemplate(ctx context.Context) (model.NotifyTemplate, error)
	ListNotifyTemplates(ctx context.Context, opts ListNotifyTemplateOptions) ([]model.NotifyTemplate, error)
	UpdateNotifyTemplate(ctx context.Context, id int64, tmpl model.NotifyTemplate) error
	// SetDefaultNotifyTemplate marks id as the sole global default template.
	SetDefaultNotifyTemplate(ctx context.Context, id int64) error
	DeleteNotifyTemplate(ctx context.Context, id int64) error

	// Notify URI masking
	MaskNotifyURI(protocol, uri string) string
}

var Database Adapter

func Init() {
	Store = storeObj{}
	pgAdapter, ok := availableAdapters["postgres"]
	if !ok {
		flog.Fatal("postgres adapter not available - check build tags")
	}
	Database = pgAdapter
}

// ---------------------------------------------------------------------------
// AuditStore
// ---------------------------------------------------------------------------

type AuditStore struct {
	client *gen.Client
}

func NewAuditStore(client *gen.Client) *AuditStore {
	return &AuditStore{client: client}
}

// Record writes an audit entry to persistent storage.
// If the store or client is nil, the call is silently skipped.
// Audit write failures are logged and do not propagate to the caller.
// Sensitive fields in entry.Request are redacted before storage.
func (s *AuditStore) Record(ctx context.Context, entry audit.Entry) error {
	if s == nil || s.client == nil {
		return nil
	}
	actorUID := ""
	details := map[string]any{}
	if entry.Subject != nil {
		actorUID = entry.Subject.SubjectType + ":" + entry.Subject.SubjectID
		details["subject_type"] = entry.Subject.SubjectType
		details["subject_id"] = entry.Subject.SubjectID
		details["uid"] = entry.Subject.UID
		details["ip_address"] = entry.Subject.IPAddress
		details["user_agent"] = entry.Subject.UserAgent
	}
	if entry.Request != nil {
		details["request"] = sanitizeAuditValue(entry.Request)
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.Target.Type).
		SetTargetID(entry.Target.ID).
		SetActorUID(actorUID).
		SetDetails(details).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		flog.Warn("audit write failed: %v", err)
		return nil
	}
	return nil
}

// RecordSuccess writes a success audit entry.
func (s *AuditStore) RecordSuccess(ctx context.Context, entry audit.Entry) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "success")
	return s.Record(ctx, e)
}

// RecordFailure writes a failure audit entry with the error message.
func (s *AuditStore) RecordFailure(ctx context.Context, entry audit.Entry, err error) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "failed")
	if err != nil {
		e.Request = wrapResult(e.Request, "error", err.Error())
	}
	return s.Record(ctx, e)
}

// RecordRejected writes a rejected audit entry with the reason.
func (s *AuditStore) RecordRejected(ctx context.Context, entry audit.Entry, reason string) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "rejected")
	e.Request = wrapResult(e.Request, "error", reason)
	return s.Record(ctx, e)
}

func wrapResult(request any, key, value string) map[string]any {
	m := map[string]any{key: value}
	if request != nil {
		if existing, ok := request.(map[string]any); ok {
			for k, v := range existing {
				if _, exists := m[k]; !exists {
					m[k] = v
				}
			}
		}
	}
	return m
}

// auditSensitiveKeys lists request field names that are redacted from audit
// logs. All comparisons are case-insensitive (lowercase).
var auditSensitiveKeys = map[string]struct{}{
	"password":      {},
	"passwd":        {},
	"secret":        {},
	"token":         {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"cookie":        {},
	"private_key":   {},
	"access_token":  {},
	"auth":          {},
	"credential":    {},
	"signature":     {},
}

// sanitizeAuditValue recursively redacts sensitive fields from audit data.
// Map keys matching auditSensitiveKeys are replaced with "[redacted]".
// Non-map values are returned unchanged.
func sanitizeAuditValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			if _, sensitive := auditSensitiveKeys[strings.ToLower(k)]; sensitive {
				out[k] = "[redacted]"
				continue
			}
			out[k] = sanitizeAuditValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = sanitizeAuditValue(vv)
		}
		return out
	default:
		return v
	}
}

// PipelineRunInfo is a lightweight view of a pipeline run for event matching display.
type PipelineRunInfo struct {
	PipelineName  string
	EventID       string
	Status        string
	TriggerSource string
}

// ---------------------------------------------------------------------------
// EventStore
// ---------------------------------------------------------------------------

type EventStore struct {
	client *gen.Client
}

func NewEventStore(client *gen.Client) *EventStore {
	return &EventStore{client: client}
}

func (s *EventStore) AppendDataEvent(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	c := s.client.DataEvent.Create().
		SetEventID(event.EventID).
		SetEventType(event.EventType).
		SetSource(event.Source).
		SetCapability(event.Capability).
		SetOperation(event.Operation).
		SetApp(event.App).
		SetEntityID(event.EntityID).
		SetIdempotencyKey(event.IdempotencyKey).
		SetUID(event.UID).
		SetTopic(event.Topic).
		SetCreatedAt(time.Now())
	if event.Data != nil {
		c = c.SetData(map[string]any(event.Data))
	}
	if event.Tags != nil {
		c = c.SetTags(map[string]any(event.Tags))
	}
	_, err := c.Save(ctx)
	if err == nil && event.Source != "" {
		types.EventFilterCache.SetSource(event.Source)
	}
	if err == nil && event.EventType != "" {
		types.EventFilterCache.SetEventType(event.EventType)
	}
	return err
}

func (s *EventStore) AppendEventOutbox(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	payload := map[string]any{
		"event_id":        event.EventID,
		"event_type":      event.EventType,
		"source":          event.Source,
		"capability":      event.Capability,
		"operation":       event.Operation,
		"app":             event.App,
		"entity_id":       event.EntityID,
		"idempotency_key": event.IdempotencyKey,
		"uid":             event.UID,
		"topic":           event.Topic,
	}
	if event.Tags != nil {
		payload["tags"] = map[string]any(event.Tags)
	}
	_, err := s.client.EventOutbox.Create().
		SetEventID(event.EventID).
		SetPayload(payload).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *EventStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventOutbox.Update().
		Where(eventoutbox.EventID(eventID)).
		SetPublished(true).
		Save(ctx)
	return err
}

// ListDataEventsOptions holds filters and pagination for listing data events.
type ListDataEventsOptions struct {
	Limit        int        // max 100, default 20
	Offset       int        // page offset for offset-based pagination
	Cursor       string     // opaque CreatedAt cursor (backward compatible)
	Source       string     // filter by source, empty = all
	EventType    string     // filter by event type, empty = all
	Webhook      bool       // if true, only events where data->>'_webhook_method' IS NOT NULL
	Search       string     // ILIKE match against source and data::text
	PipelineName string     // filter events that triggered a specific pipeline
	TimeStart    *time.Time // created_at >= TimeStart
	TimeEnd      *time.Time // created_at <= TimeEnd
}

// ListDataEvents returns paginated data_events ordered by created_at DESC.
// Supports offset-based pagination (when Offset > 0) and cursor-based (backward compatible).
func (s *EventStore) ListDataEvents(ctx context.Context, opts ListDataEventsOptions) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := applyDataEventFilters(s.client, opts)

	// Offset-based pagination (mutually exclusive with cursor)
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset).Limit(opts.Limit)
		events, err := q.All(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("list data events: %w", err)
		}
		return events, "", nil
	}

	// Cursor-based pagination (backward compatible)
	q = q.Limit(opts.Limit + 1)
	if opts.Cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", opts.Cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list data events: %w", err)
	}

	var nextCursor string
	if len(events) > opts.Limit {
		nextCursor = events[opts.Limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		events = events[:opts.Limit]
	}

	return events, nextCursor, nil
}

// applyDataEventFilters applies all filter options from ListDataEventsOptions
// to a new base query ordered by created_at DESC.
func applyDataEventFilters(client *gen.Client, opts ListDataEventsOptions) *gen.DataEventQuery {
	q := client.DataEvent.Query().
		Order(dataevent.ByCreatedAt(sql.OrderDesc()))

	if opts.Source != "" {
		q = q.Where(dataevent.Source(opts.Source))
	}
	if opts.EventType != "" {
		q = q.Where(dataevent.EventType(opts.EventType))
	}
	if opts.Webhook {
		q = q.Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP("data->>'_webhook_method' IS NOT NULL"))
		})
	}
	if opts.Search != "" {
		q = q.Where(sql.OrPredicates(
			func(s *sql.Selector) { s.Where(sql.ContainsFold("source", opts.Search)) },
			func(s *sql.Selector) {
				switch s.Dialect() {
				case dialect.Postgres:
					s.Where(sql.ExprP("CAST(data AS TEXT) ILIKE '%' || $1 || '%'", opts.Search))
				default:
					s.Where(sql.ExprP("LOWER(CAST(data AS TEXT)) LIKE LOWER('%' || $1 || '%')", opts.Search))
				}
			},
		))
	}
	if opts.PipelineName != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"event_id IN (SELECT event_id FROM pipeline_runs WHERE pipeline_name = $1)",
				opts.PipelineName,
			))
		})
	}
	if opts.TimeStart != nil {
		q = q.Where(dataevent.CreatedAtGTE(*opts.TimeStart))
	}
	if opts.TimeEnd != nil {
		q = q.Where(dataevent.CreatedAtLTE(*opts.TimeEnd))
	}

	return q
}

// CountDataEvents returns the total number of data_events matching the given filters.
// Uses the same filter predicates as ListDataEvents without pagination.
func (s *EventStore) CountDataEvents(ctx context.Context, opts ListDataEventsOptions) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}

	q := applyDataEventFilters(s.client, opts)

	count, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count data events: %w", err)
	}

	return int64(count), nil
}

// ListDistinctEventPipelineNames returns distinct pipeline names from pipeline_runs
// that have matched events, ordered alphabetically.
func (s *EventStore) ListDistinctEventPipelineNames(ctx context.Context) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	rows, err := s.client.PipelineRun.Query().
		GroupBy(pipelinerun.FieldPipelineName).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct pipeline names: %w", err)
	}

	slices.Sort(rows)
	return rows, nil
}

// ListDistinctEventSources returns unique source values from data_events
// created within the given duration (e.g. 30*24*time.Hour for last 30 days).
func (s *EventStore) ListDistinctEventSources(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	sources, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldSource).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event sources: %w", err)
	}
	return sources, nil
}

// ListDistinctEventTypes returns unique event_type values from data_events
// created within the given duration.
func (s *EventStore) ListDistinctEventTypes(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	distinctTypes, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldEventType).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event types: %w", err)
	}
	return distinctTypes, nil
}

// GetDataEventByEventID looks up a single data event by its event_id.
func (s *EventStore) GetDataEventByEventID(ctx context.Context, eventID string) (*gen.DataEvent, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	e, err := s.client.DataEvent.Query().
		Where(dataevent.EventID(eventID)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get data event by id: %w", err)
	}
	return e, nil
}

// DeleteDataEventsOlderThan deletes data_events with created_at before cutoff
// and related history (pipeline step runs, pipeline runs, event consumptions,
// event outbox rows, and resource links that reference those events).
// Returns the number of deleted data_events rows.
func (s *EventStore) DeleteDataEventsOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	const batchSize = 500
	total := 0
	for {
		n, err := s.deleteDataEventsBatch(ctx, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		total += n
		if n < batchSize {
			return total, nil
		}
	}
}

// deleteDataEventsBatch purges up to limit old data_events and their dependents
// in a single transaction.
func (s *EventStore) deleteDataEventsBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	events, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtLT(cutoff)).
		Order(dataevent.ByCreatedAt()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("list old data events: %w", err)
	}
	if len(events) == 0 {
		return 0, nil
	}

	eventIDs := make([]string, len(events))
	eventPKs := make([]int64, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
		eventPKs[i] = e.ID
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retention tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := purgeEventHistory(ctx, tx, eventIDs); err != nil {
		return 0, err
	}

	n, err := tx.DataEvent.Delete().
		Where(dataevent.IDIn(eventPKs...)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete old data events: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit retention tx: %w", err)
	}
	committed = true
	return n, nil
}

// purgeEventHistory removes pipeline and delivery rows that reference eventIDs.
func purgeEventHistory(ctx context.Context, tx *gen.Tx, eventIDs []string) error {
	runs, err := tx.PipelineRun.Query().
		Where(pipelinerun.EventIDIn(eventIDs...)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list pipeline runs for retention: %w", err)
	}
	if err := purgePipelineRuns(ctx, tx, runs); err != nil {
		return err
	}
	if _, err := tx.ResourceLink.Delete().
		Where(resourcelink.Or(
			resourcelink.SourceEventIDIn(eventIDs...),
			resourcelink.TargetEventIDIn(eventIDs...),
		)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete resource links by event: %w", err)
	}
	if _, err := tx.EventConsumption.Delete().
		Where(eventconsumption.EventIDIn(eventIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete event consumptions: %w", err)
	}
	if _, err := tx.EventOutbox.Delete().
		Where(eventoutbox.EventIDIn(eventIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete event outbox: %w", err)
	}
	return nil
}

// purgePipelineRuns deletes step runs, resource links, and pipeline runs for the given runs.
func purgePipelineRuns(ctx context.Context, tx *gen.Tx, runs []*gen.PipelineRun) error {
	if len(runs) == 0 {
		return nil
	}
	runIDs := make([]int64, len(runs))
	for i, r := range runs {
		runIDs[i] = r.ID
	}
	if _, err := tx.PipelineStepRun.Delete().
		Where(pipelinesteprun.PipelineRunIDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete pipeline step runs: %w", err)
	}
	if _, err := tx.ResourceLink.Delete().
		Where(resourcelink.PipelineRunIDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete resource links by run: %w", err)
	}
	if _, err := tx.PipelineRun.Delete().
		Where(pipelinerun.IDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete pipeline runs: %w", err)
	}
	return nil
}

// GetPipelineRunsForEvents batch-looks up pipeline runs for the given event IDs.
// Returns a map of eventID -> []PipelineRunInfo.
func (s *EventStore) GetPipelineRunsForEvents(ctx context.Context, eventIDs []string) (map[string][]PipelineRunInfo, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.EventIDIn(eventIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pipeline runs for events: %w", err)
	}
	result := make(map[string][]PipelineRunInfo, len(runs))
	for _, r := range runs {
		info := PipelineRunInfo{
			PipelineName:  r.PipelineName,
			EventID:       r.EventID,
			Status:        fmt.Sprintf("%d", r.Status),
			TriggerSource: string(r.TriggerSource),
		}
		result[r.EventID] = append(result[r.EventID], info)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// PipelineStore
// ---------------------------------------------------------------------------

// PipelineStore persists pipeline definitions, runs, step runs, and event consumptions.
type PipelineStore struct {
	client *gen.Client
}

func NewPipelineStore(client *gen.Client) *PipelineStore {
	return &PipelineStore{client: client}
}

func (s *PipelineStore) CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	run, err := s.client.PipelineRun.Create().
		SetPipelineName(pipelineName).
		SetEventID(eventID).
		SetEventType(eventType).
		SetTriggerSource(pipelinerun.TriggerSource(triggerSource)).
		SetStatus(int(schema.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *PipelineStore) UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineRun.UpdateOneID(runID).
		SetStatus(int(status)).
		SetCompletedAt(time.Now())
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params map[string]any, attempt int) (*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	sr, err := s.client.PipelineStepRun.Create().
		SetPipelineRunID(runID).
		SetStepName(stepName).
		SetCapability(capability).
		SetOperation(operation).
		SetParams(map[string]any(params)).
		SetAttempt(attempt).
		SetStatus(int(schema.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *PipelineStore) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineStepRun.UpdateOneID(stepRunID).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == int(schema.PipelineDone) || status == int(schema.PipelineCancel) {
		now := time.Now()
		upd = upd.SetCompletedAt(now)
	}
	if result != nil {
		upd = upd.SetResult(result)
	}
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) RecordConsumption(ctx context.Context, consumerName, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventConsumption.Create().
		SetConsumerName(consumerName).
		SetEventID(eventID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error) {
	if s == nil || s.client == nil {
		return false, nil
	}
	count, err := s.client.EventConsumption.Query().
		Where(
			eventconsumption.ConsumerName(consumerName),
			eventconsumption.EventID(eventID),
		).
		Count(ctx)
	return count > 0, err
}

// SaveCheckpoint persists the intermediate pipeline run state.
func (s *PipelineStore) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	var cp map[string]any
	if err := sonic.Unmarshal(raw, &cp); err != nil {
		return err
	}
	_, err = s.client.PipelineRun.UpdateOneID(runID).
		SetCheckpointData(cp).
		Save(ctx)
	return err
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running pipeline.
func (s *PipelineStore) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.PipelineRun.UpdateOneID(runID).
		SetLastHeartbeat(time.Now()).
		Save(ctx)
	return err
}

// GetIncompleteRuns returns pipeline runs that are in Start state and may need recovery.
func (s *PipelineStore) GetIncompleteRuns(ctx context.Context) ([]*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Status(int(schema.PipelineStart))).
		Order(pipelinerun.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

// GetCheckpoint loads the checkpoint data for a pipeline run.
func (s *PipelineStore) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Select(pipelinerun.FieldCheckpointData).
		Only(ctx)
	if err != nil {
		return err
	}
	if run.CheckpointData == nil {
		return nil
	}
	raw, err := sonic.Marshal(run.CheckpointData)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, target)
}

// GetRun returns a pipeline run by ID.
func (s *PipelineStore) GetRun(ctx context.Context, runID int64) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// RecordResourceLink inserts a resource link with UPSERT semantics.
func (s *PipelineStore) RecordResourceLink(ctx context.Context, link *gen.ResourceLink) error {
	if s == nil || s.client == nil {
		return nil
	}
	err := s.client.ResourceLink.Create().
		SetSourceEventID(link.SourceEventID).
		SetTargetEventID(link.TargetEventID).
		SetSourceApp(link.SourceApp).
		SetTargetApp(link.TargetApp).
		SetSourceCapability(link.SourceCapability).
		SetTargetCapability(link.TargetCapability).
		SetSourceEntityID(link.SourceEntityID).
		SetTargetEntityID(link.TargetEntityID).
		SetPipelineRunID(link.PipelineRunID).
		SetPipelineName(link.PipelineName).
		SetCreatedAt(time.Now()).
		OnConflictColumns(
			resourcelink.FieldSourceEventID,
			resourcelink.FieldTargetEventID,
		).
		Ignore().
		Exec(ctx)
	return err
}

// CreateDefinition creates a new pipeline definition with initial yaml_draft and version 1.
// createdBy is the Web UI user UID that created the pipeline (may be empty in tests).
func (s *PipelineStore) CreateDefinition(ctx context.Context, name, description, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.PipelineDefinition.Create().
		SetName(name).
		SetDescription(description).
		SetYamlDraft("").
		SetNillableYamlPublished(nil).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedBy(strings.TrimSpace(createdBy)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if gen.IsConstraintError(err) {
			return fmt.Errorf("pipeline %q %w", name, types.ErrAlreadyExists)
		}
		return err
	}
	return nil
}

// EnsureDefinitionCreatedBy sets created_by when it is currently empty.
// Used to backfill owner UID for pipelines created before the field existed.
func (s *PipelineStore) EnsureDefinitionCreatedBy(ctx context.Context, name, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return nil
	}
	_, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.CreatedByEQ(""),
		).
		SetCreatedBy(createdBy).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("ensure pipeline created_by: %w", err)
	}
	return nil
}

// GetDefinitionByName returns a pipeline definition by name.
func (s *PipelineStore) GetDefinitionByName(ctx context.Context, name string) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ListDefinitionVersions returns all published version snapshots for a pipeline,
// ordered by version descending (newest first).
func (s *PipelineStore) ListDefinitionVersions(ctx context.Context, name string) ([]*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinitionVersion.Query().
		Where(pipelinedefinitionversion.PipelineName(name)).
		Order(gen.Desc(pipelinedefinitionversion.FieldVersion)).
		All(ctx)
}

// GetDefinitionVersion returns a single version snapshot by pipeline name and version number.
func (s *PipelineStore) GetDefinitionVersion(ctx context.Context, name string, version int) (*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinitionVersion.Query().
		Where(
			pipelinedefinitionversion.PipelineName(name),
			pipelinedefinitionversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ListDefinitions returns all pipeline definitions ordered by updated_at desc.
func (s *PipelineStore) ListDefinitions(ctx context.Context) ([]*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinition.Query().
		Order(gen.Desc(pipelinedefinition.FieldUpdatedAt)).
		All(ctx)
}

// UpdateDefinitionDraft updates the yaml_draft with atomic optimistic locking.
// Uses conditional UPDATE WHERE version=X. Returns ErrConflict if no row matched.
func (s *PipelineStore) UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlDraft(yamlDraft).
		SetVersion(version + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}
	return s.GetDefinitionByName(ctx, name)
}

// PublishDefinition copies yaml_draft to yaml_published with atomic optimistic locking.
// Also inserts a version snapshot into pipeline_definition_versions.
func (s *PipelineStore) PublishDefinition(ctx context.Context, name string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.YamlDraft == "" {
		return nil, types.ErrConflict
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlPublished(def.YamlDraft).
		SetVersion(version + 1).
		SetStatus("published").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}

	if _, err := s.client.PipelineDefinitionVersion.Create().
		SetPipelineName(name).
		SetVersion(version + 1).
		SetYaml(def.YamlDraft).
		SetCreatedAt(time.Now()).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("publish: insert version snapshot: %w", err)
	}

	return s.GetDefinitionByName(ctx, name)
}

// SetDefinitionEnabled toggles the top-level enabled flag in draft and published YAML.
// Only published pipelines can be paused at runtime.
func (s *PipelineStore) SetDefinitionEnabled(ctx context.Context, name string, enabled bool) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.Status != pipelinedefinition.StatusPublished || def.YamlPublished == nil || *def.YamlPublished == "" {
		return nil, fmt.Errorf("pipeline %q is not published: %w", name, types.ErrInvalidArgument)
	}

	newPublished, err := pipeline.SetEnabledInYAML(*def.YamlPublished, enabled)
	if err != nil {
		return nil, fmt.Errorf("set enabled on published yaml: %w", err)
	}

	update := s.client.PipelineDefinition.Update().
		Where(pipelinedefinition.Name(name)).
		SetYamlPublished(newPublished).
		SetUpdatedAt(time.Now())

	if def.YamlDraft != "" {
		newDraft, draftErr := pipeline.SetEnabledInYAML(def.YamlDraft, enabled)
		if draftErr != nil {
			return nil, fmt.Errorf("set enabled on draft yaml: %w", draftErr)
		}
		update = update.SetYamlDraft(newDraft)
	}

	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	return s.GetDefinitionByName(ctx, name)
}

// DeleteDefinitionByName removes a pipeline definition and its associated runs.
// Returns the number of pipeline runs that were deleted.
func (s *PipelineStore) DeleteDefinitionByName(ctx context.Context, name string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	runCount, err := s.client.PipelineRun.Delete().
		Where(pipelinerun.PipelineName(name)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete runs for %s: %w", name, err)
	}
	_, err = s.client.PipelineDefinition.Delete().
		Where(pipelinedefinition.Name(name)).
		Exec(ctx)
	if err != nil {
		return int64(runCount), fmt.Errorf("delete definition %s: %w", name, err)
	}
	return int64(runCount), nil
}

// pipelineRunByParentName matches exact parent name and compound trigger engine names.
func pipelineRunByParentName(parentName string) predicate.PipelineRun {
	return pipelinerun.Or(
		pipelinerun.PipelineName(parentName),
		pipelinerun.PipelineNameHasPrefix(parentName+"__trigger_"),
	)
}

// GetRunsByParentName returns pipeline runs matching a parent pipeline name.
// Matches both exact name and compound trigger names (name__trigger_*).
func (s *PipelineStore) GetRunsByParentName(ctx context.Context, parentName string) ([]*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineRun.Query().
		Where(pipelineRunByParentName(parentName)).
		Order(gen.Desc(pipelinerun.FieldCreatedAt)).
		Limit(100).
		All(ctx)
}

// LatestRunStartedAtByParentNames returns the latest started_at for each parent pipeline name.
// Matches exact pipeline_name and compound trigger names (name__trigger_*).
// Names without runs are omitted from the result.
func (s *PipelineStore) LatestRunStartedAtByParentNames(ctx context.Context, names []string) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	if s == nil || s.client == nil || len(names) == 0 {
		return result, nil
	}
	preds := make([]predicate.PipelineRun, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		preds = append(preds, pipelineRunByParentName(name))
	}
	if len(preds) == 0 {
		return result, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Or(preds...)).
		Select(pipelinerun.FieldPipelineName, pipelinerun.FieldStartedAt).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest run started_at by parent names: %w", err)
	}
	for _, run := range runs {
		parent := matchParentPipelineName(run.PipelineName, names)
		if parent == "" {
			continue
		}
		if prev, ok := result[parent]; !ok || run.StartedAt.After(prev) {
			result[parent] = run.StartedAt
		}
	}
	return result, nil
}

// matchParentPipelineName maps a run pipeline_name back to a parent definition name.
func matchParentPipelineName(runName string, parents []string) string {
	best := ""
	for _, parent := range parents {
		if parent == "" {
			continue
		}
		if runName == parent {
			return parent
		}
		if strings.HasPrefix(runName, parent+"__trigger_") && len(parent) > len(best) {
			best = parent
		}
	}
	return best
}

// GetStepRunsByRunID returns all step runs for a given pipeline run, ordered by ID.
func (s *PipelineStore) GetStepRunsByRunID(ctx context.Context, runID int64) ([]*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineStepRun.Query().
		Where(pipelinesteprun.PipelineRunIDEQ(runID)).
		Order(gen.Asc(pipelinesteprun.FieldID)).
		Limit(200).
		All(ctx)
}

// GetRunByID returns a pipeline run by its database ID.
func (s *PipelineStore) GetRunByID(ctx context.Context, id int64) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	run, err := s.client.PipelineRun.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return run, nil
}

// ListStepRunsByRunID returns all step runs for a pipeline run, ordered by creation time.
func (s *PipelineStore) ListStepRunsByRunID(ctx context.Context, runID int64) ([]*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	steps, err := s.client.PipelineStepRun.Query().
		Where(pipelinesteprun.PipelineRunIDEQ(runID)).
		Order(gen.Asc(pipelinesteprun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

// ListPublishedDefinitions returns all pipeline definitions that are published
// and have a non-nil yaml_published field.
func (s *PipelineStore) ListPublishedDefinitions(ctx context.Context) ([]pipeline.DefinitionRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.PipelineDefinition.Query().
		Where(
			pipelinedefinition.StatusEQ(pipelinedefinition.Status("published")),
			pipelinedefinition.YamlPublishedNotNil(),
		).
		Order(gen.Asc(pipelinedefinition.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published pipeline definitions: %w", err)
	}
	records := make([]pipeline.DefinitionRecord, 0, len(rows))
	for _, row := range rows {
		yamlContent := ""
		if row.YamlPublished != nil {
			yamlContent = *row.YamlPublished
		}
		records = append(records, pipeline.DefinitionRecord{
			Name:        row.Name,
			Description: row.Description,
			YAML:        yamlContent,
			CreatedBy:   row.CreatedBy,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return records, nil
}

// PipelineStats returns aggregated pipeline run statistics for chart rendering.
// name empty = all pipelines. since zero = no time filter. groupBy = "day"|"week"|"month".
func (s *PipelineStore) PipelineStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.PipelineStats, error) {
	if s == nil || s.client == nil {
		return emptyPipelineStats(), nil
	}
	stats := &types.PipelineStats{}

	var err error
	stats.Summary, err = s.loadPipelineStatsSummary(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}
	stats.SuccessRateTrend, err = s.loadSuccessRate(ctx, name, since, groupBy)
	if err != nil {
		return nil, fmt.Errorf("success rate: %w", err)
	}
	stats.DurationDistribution.Pipeline, err = s.loadDurationBuckets(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("pipeline duration: %w", err)
	}
	stats.DurationDistribution.Step, err = s.loadStepDurationBuckets(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("step duration: %w", err)
	}
	stats.TriggerSourcePie, err = s.loadTriggerSources(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("trigger sources: %w", err)
	}
	return stats, nil
}

// loadPipelineStatsSummary returns headline counters for the pipelines overview.
func (s *PipelineStore) loadPipelineStatsSummary(ctx context.Context, name string, since time.Time) (types.PipelineStatsSummary, error) {
	summary := types.PipelineStatsSummary{}
	if name == "" {
		count, err := s.client.PipelineDefinition.Query().Count(ctx)
		if err != nil {
			return summary, err
		}
		summary.TotalPipelines = int64(count)
	}

	successful, err := s.countCompletedRunsByStatus(ctx, name, since, int(schema.PipelineDone))
	if err != nil {
		return summary, err
	}
	failed, err := s.countCompletedRunsByStatus(ctx, name, since, int(schema.PipelineFailed))
	if err != nil {
		return summary, err
	}
	summary.SuccessfulRuns = successful
	summary.FailedRuns = failed
	return summary, nil
}

// countCompletedRunsByStatus counts completed runs filtered by pipeline, time range, and status.
func (s *PipelineStore) countCompletedRunsByStatus(ctx context.Context, name string, since time.Time, status int) (int64, error) {
	q := s.client.PipelineRun.Query().
		Where(
			pipelinerun.CompletedAtNotNil(),
			pipelinerun.StatusEQ(status),
		)
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}
	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

// loadSuccessRate fetches completed runs and computes success rate points in Go.
// Ent v0.14.6 does not expose Modify() on query builders, so custom SQL GROUP BY
// expressions are handled via in-memory aggregation after fetching raw data.
func (s *PipelineStore) loadSuccessRate(ctx context.Context, name string, since time.Time, groupBy string) ([]types.SuccessRatePoint, error) {
	runs, err := s.fetchCompletedRuns(ctx, name, since)
	if err != nil {
		return nil, err
	}
	return computeSuccessRate(runs, groupBy), nil
}

// fetchCompletedRuns returns completed pipeline runs filtered by name and since time.
func (s *PipelineStore) fetchCompletedRuns(ctx context.Context, name string, since time.Time) ([]*gen.PipelineRun, error) {
	q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}
	return q.All(ctx)
}

// computeSuccessRate aggregates completed runs into success rate points grouped by date.
func computeSuccessRate(runs []*gen.PipelineRun, groupBy string) []types.SuccessRatePoint {
	type dayStats struct {
		total   int64
		success int64
	}
	buckets := make(map[string]*dayStats)
	for _, r := range runs {
		if r.CompletedAt == nil {
			continue
		}
		key := dateGroupKey(*r.CompletedAt, groupBy)
		if buckets[key] == nil {
			buckets[key] = &dayStats{}
		}
		buckets[key].total++
		if r.Status == int(schema.PipelineDone) {
			buckets[key].success++
		}
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	points := make([]types.SuccessRatePoint, 0, len(keys))
	for _, k := range keys {
		s := buckets[k]
		rate := float64(0)
		if s.total > 0 {
			rate = float64(s.success) / float64(s.total)
		}
		points = append(points, types.SuccessRatePoint{
			Date: k, Total: s.total, Success: s.success, Rate: rate,
		})
	}
	if points == nil {
		points = []types.SuccessRatePoint{}
	}
	return points
}

// dateGroupKey returns a date grouping key based on the requested granularity.
func dateGroupKey(t time.Time, groupBy string) string {
	switch groupBy {
	case "week":
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}

// loadDurationBuckets fetches completed runs and buckets by duration in Go.
func (s *PipelineStore) loadDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
	q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}

	runs, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := emptyDurationBuckets()
	for _, r := range runs {
		if r.StartedAt.IsZero() || r.CompletedAt == nil {
			continue
		}
		dur := r.CompletedAt.Sub(r.StartedAt)
		switch {
		case dur < time.Second:
			result[0].Count++
		case dur < 5*time.Second:
			result[1].Count++
		case dur < 30*time.Second:
			result[2].Count++
		default:
			result[3].Count++
		}
	}
	return result, nil
}

// loadStepDurationBuckets fetches completed step runs and buckets by duration.
func (s *PipelineStore) loadStepDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
	q := s.client.PipelineStepRun.Query().Where(pipelinesteprun.CompletedAtNotNil())
	if name != "" {
		runIDs, err := s.client.PipelineRun.Query().
			Where(pipelineRunByParentName(name)).
			IDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(runIDs) == 0 {
			return emptyDurationBuckets(), nil
		}
		q = q.Where(pipelinesteprun.PipelineRunIDIn(runIDs...))
	}
	if !since.IsZero() {
		q = q.Where(pipelinesteprun.StartedAtGTE(since))
	}

	steps, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := emptyDurationBuckets()
	for _, st := range steps {
		if st.StartedAt.IsZero() || st.CompletedAt == nil {
			continue
		}
		dur := st.CompletedAt.Sub(st.StartedAt)
		switch {
		case dur < time.Second:
			result[0].Count++
		case dur < 5*time.Second:
			result[1].Count++
		case dur < 30*time.Second:
			result[2].Count++
		default:
			result[3].Count++
		}
	}
	return result, nil
}

// loadTriggerSources counts runs grouped by trigger_source using ent GroupBy.
func (s *PipelineStore) loadTriggerSources(ctx context.Context, name string, since time.Time) ([]types.TriggerSourceCount, error) {
	q := s.client.PipelineRun.Query()
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}

	type row struct {
		Source string `sql:"trigger_source"`
		Count  int64  `sql:"count"`
	}
	var rows []row

	err := q.GroupBy(pipelinerun.FieldTriggerSource).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	result := map[string]int64{"event": 0, "webhook": 0, "cron": 0, "manual": 0}
	for _, r := range rows {
		result[r.Source] = r.Count
	}
	return []types.TriggerSourceCount{
		{Source: "event", Count: result["event"]},
		{Source: "webhook", Count: result["webhook"]},
		{Source: "cron", Count: result["cron"]},
		{Source: "manual", Count: result["manual"]},
	}, nil
}

func emptyPipelineStats() *types.PipelineStats {
	return &types.PipelineStats{
		Summary: types.PipelineStatsSummary{},
		TriggerSourcePie: []types.TriggerSourceCount{
			{Source: "event"}, {Source: "webhook"}, {Source: "cron"}, {Source: "manual"},
		},
		DurationDistribution: types.DurationDistribution{
			Pipeline: emptyDurationBuckets(),
			Step:     emptyDurationBuckets(),
		},
	}
}

func emptyDurationBuckets() []types.DurationEntry {
	return []types.DurationEntry{
		{Bucket: "0-1s"}, {Bucket: "1-5s"}, {Bucket: "5-30s"}, {Bucket: "30s+"},
	}
}

// ---------------------------------------------------------------------------
// PollingStateStore
// ---------------------------------------------------------------------------

// PollingStateStore persists polling state entries for the provider event source framework.
type PollingStateStore struct {
	client *gen.Client
}

// NewPollingStateStore returns a PollingStateStore backed by the given Ent client.
func NewPollingStateStore(client *gen.Client) *PollingStateStore {
	return &PollingStateStore{client: client}
}

// PollingStateEntry represents a single persisted polling state row.
type PollingStateEntry struct {
	Cursor      string
	KnownHashes map[string]string
	UpdatedAt   any
}

// LoadAll loads all polling state entries from the database.
func (s *PollingStateStore) LoadAll(ctx context.Context) (map[string]PollingStateEntry, error) {
	if s == nil || s.client == nil {
		return make(map[string]PollingStateEntry), nil
	}
	rows, err := s.client.PollingState.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]PollingStateEntry, len(rows))
	for _, row := range rows {
		result[row.ResourceName] = PollingStateEntry{
			Cursor:      row.Cursor,
			KnownHashes: row.KnownHashes,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return result, nil
}

// Save upserts a polling state entry for the given resource.
// If an entry with the same resource name already exists, it is updated; otherwise a new one is created.
func (s *PollingStateStore) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if s == nil || s.client == nil {
		return nil
	}
	if knownHashes == nil {
		knownHashes = make(map[string]string)
	}
	existing, err := s.client.PollingState.Query().
		Where(pollingstate.ResourceName(resourceName)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return err
	}
	if existing != nil {
		_, err = s.client.PollingState.UpdateOne(existing).
			SetCursor(cursor).
			SetKnownHashes(knownHashes).
			Save(ctx)
		return err
	}
	_, err = s.client.PollingState.Create().
		SetResourceName(resourceName).
		SetCursor(cursor).
		SetKnownHashes(knownHashes).
		Save(ctx)
	return err
}

// ---------------------------------------------------------------------------
// WorkflowRunStore
// ---------------------------------------------------------------------------

// WorkflowRunStore persists workflow runs, step runs, and checkpoint data.
type WorkflowRunStore struct {
	client *gen.Client
}

// NewWorkflowRunStore creates a WorkflowRunStore backed by the given ent client.
func NewWorkflowRunStore(client *gen.Client) *WorkflowRunStore {
	return &WorkflowRunStore{client: client}
}

// CreateRun inserts a new workflow run record.
func (s *WorkflowRunStore) CreateRun(ctx context.Context, workflowName, workflowFile, triggerType string, triggerInfo, inputParams map[string]any) (*gen.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	wr, err := s.client.WorkflowRun.Create().
		SetWorkflowName(workflowName).
		SetWorkflowFile(workflowFile).
		SetStatus(int(schema.WorkflowRunRunning)).
		SetTriggerType(triggerType).
		SetTriggerInfo(map[string]any(triggerInfo)).
		SetInputParams(map[string]any(inputParams)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return wr, nil
}

// UpdateRunStatus updates the status, error, and completed_at of a workflow run.
func (s *WorkflowRunStore) UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	u := s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetStatus(int(status)).
		SetCompletedAt(now)
	if errMsg != "" {
		u = u.SetError(errMsg)
	}
	return u.Exec(ctx)
}

// CreateStepRun inserts a new workflow step run record.
func (s *WorkflowRunStore) CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params map[string]any, attempt int) (*gen.WorkflowStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	sr, err := s.client.WorkflowStepRun.Create().
		SetWorkflowRunID(runID).
		SetStepID(stepID).
		SetStepName(stepName).
		SetAction(action).
		SetActionType(actionType).
		SetParams(map[string]any(params)).
		SetAttempt(attempt).
		SetStatus(int(schema.WorkflowRunRunning)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

// UpdateStepRun updates the status, result, error, and attempt count of a workflow step run.
// completed_at is only set for terminal states (Done, Failed).
func (s *WorkflowRunStore) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	u := s.client.WorkflowStepRun.Update().
		Where(workflowsteprun.IDEQ(stepRunID)).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == int(schema.WorkflowRunDone) || status == int(schema.WorkflowRunFailed) {
		u = u.SetCompletedAt(time.Now())
	}
	if result != nil {
		u = u.SetResult(map[string]any(result))
	}
	if errMsg != "" {
		u = u.SetError(errMsg)
	}
	return u.Exec(ctx)
}

// SaveCheckpoint persists the intermediate workflow run state.
func (s *WorkflowRunStore) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	cp := schema.JSON{}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	if err := cp.Scan(raw); err != nil {
		return err
	}
	return s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetCheckpointData(map[string]any(cp)).
		Exec(ctx)
}

// GetIncompleteRuns returns workflow runs that are still running and may need recovery.
func (s *WorkflowRunStore) GetIncompleteRuns(ctx context.Context) ([]*gen.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.WorkflowRun.Query().
		Where(workflowrun.StatusEQ(int(schema.WorkflowRunRunning))).
		Order(gen.Asc(workflowrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

// GetCheckpoint loads the checkpoint data for a workflow run.
func (s *WorkflowRunStore) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
	wr, err := s.client.WorkflowRun.Query().
		Where(workflowrun.IDEQ(runID)).
		Select(workflowrun.FieldCheckpointData).
		Only(ctx)
	if err != nil {
		return err
	}
	if wr.CheckpointData == nil {
		return nil
	}
	raw, err := sonic.Marshal(wr.CheckpointData)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, target)
}

// GetRun returns a workflow run by ID.
func (s *WorkflowRunStore) GetRun(ctx context.Context, runID int64) (*gen.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	wr, err := s.client.WorkflowRun.Query().
		Where(workflowrun.IDEQ(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return wr, nil
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running workflow.
func (s *WorkflowRunStore) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.WorkflowRun.Update().
		Where(workflowrun.IDEQ(runID)).
		SetLastHeartbeat(time.Now()).
		Exec(ctx)
}

// ---------------------------------------------------------------------------
// HubStore
// ---------------------------------------------------------------------------

// HubStore persists homelab discovery data to the database.
type HubStore struct {
	client *gen.Client
}

// AppInfo is a lightweight projection of store-level app metadata.
type AppInfo struct {
	Name      string
	UpdatedAt time.Time
}

// ListApps returns all apps from the database with Name and UpdatedAt.
// When the client is nil, returns nil (safe for no-DB environments).
func (s *HubStore) ListApps(ctx context.Context) ([]AppInfo, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.App.Query().Select(app.FieldName, app.FieldUpdatedAt).Order(app.ByName()).All(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]AppInfo, len(rows))
	for i, r := range rows {
		infos[i] = AppInfo{Name: r.Name, UpdatedAt: r.UpdatedAt}
	}
	return infos, nil
}

// NewHubStore returns a HubStore backed by the given Ent client.
func NewHubStore(client *gen.Client) *HubStore {
	return &HubStore{client: client}
}

// SaveHomelabApps upserts a batch of discovered homelab apps.
// Each app is looked up by name; existing rows are updated, new rows are created.
func (s *HubStore) SaveHomelabApps(ctx context.Context, apps []homelab.App) error {
	if s == nil || s.client == nil {
		return nil
	}
	if len(apps) == 0 {
		return nil
	}

	now := time.Now()

	for _, homelabApp := range apps {
		info, err := appJSON(homelabApp)
		if err != nil {
			return err
		}

		existing, err := s.client.App.Query().
			Where(app.NameEQ(homelabApp.Name)).
			First(ctx)
		if err != nil {
			if !gen.IsNotFound(err) {
				return err
			}
			// Not found: create.
			_, createErr := s.client.App.Create().
				SetName(homelabApp.Name).
				SetPath(homelabApp.Path).
				SetStatus(string(homelabApp.Status)).
				SetDockerInfo(info).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			if createErr != nil {
				return createErr
			}
		} else {
			// Found: update.
			_, updateErr := s.client.App.UpdateOne(existing).
				SetPath(homelabApp.Path).
				SetStatus(string(homelabApp.Status)).
				SetDockerInfo(info).
				SetUpdatedAt(now).
				Save(ctx)
			if updateErr != nil {
				return updateErr
			}
		}
	}

	return nil
}

func appJSON(ha homelab.App) (schema.JSON, error) {
	raw, err := sonic.Marshal(ha)
	if err != nil {
		return nil, fmt.Errorf("marshal homelab app: %w", err)
	}
	var info schema.JSON
	if err := info.Scan(raw); err != nil {
		return nil, fmt.Errorf("scan homelab app json: %w", err)
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// ResourceChainStore
// ---------------------------------------------------------------------------

// ResourceChainStore provides query methods for resource tag and lineage lookups.
type ResourceChainStore struct {
	client *gen.Client
}

// NewResourceChainStore creates a ResourceChainStore with the given ent client.
func NewResourceChainStore(client *gen.Client) *ResourceChainStore {
	return &ResourceChainStore{client: client}
}

// FindResourcesByTag returns DataEvents matching a tag key-value pair,
// ordered by created_at descending. Supports limit + opaque cursor pagination.
func (s *ResourceChainStore) FindResourcesByTag(ctx context.Context, key, value string, limit int, cursor string) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tagJSON := fmt.Sprintf(`{%q:%q}`, key, value)
	q := s.client.DataEvent.Query().
		Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP("tags @> $1", tagJSON))
		}).
		Order(dataevent.ByCreatedAt(sql.OrderDesc())).
		Limit(limit + 1)

	if cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("find resources by tag: %w", err)
	}

	result := make([]*gen.DataEvent, len(events))
	for i, e := range events {
		result[i] = &gen.DataEvent{
			EventID:    e.EventID,
			EventType:  e.EventType,
			Source:     e.Source,
			Capability: e.Capability,
			Operation:  e.Operation,
			App:        e.App,
			EntityID:   e.EntityID,
			CreatedAt:  e.CreatedAt,
		}
		if e.Data != nil {
			result[i].Data = schema.JSON(e.Data)
		}
		if e.Tags != nil {
			result[i].Tags = schema.JSON(e.Tags)
		}
	}

	var nextCursor string
	if len(result) > limit {
		nextCursor = result[limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		result = result[:limit]
	}

	return result, nextCursor, nil
}

// FindResourceLinks returns all links involving any of the given event IDs,
// either as source or target.
func (s *ResourceChainStore) FindResourceLinks(ctx context.Context, eventIDs []string) ([]*gen.ResourceLink, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}

	links, err := s.client.ResourceLink.Query().
		Where(resourcelink.Or(
			resourcelink.SourceEventIDIn(eventIDs...),
			resourcelink.TargetEventIDIn(eventIDs...),
		)).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find resource links: %w", err)
	}

	result := make([]*gen.ResourceLink, len(links))
	for i, l := range links {
		result[i] = &gen.ResourceLink{
			ID:               l.ID,
			SourceEventID:    l.SourceEventID,
			TargetEventID:    l.TargetEventID,
			SourceApp:        l.SourceApp,
			TargetApp:        l.TargetApp,
			SourceCapability: l.SourceCapability,
			TargetCapability: l.TargetCapability,
			SourceEntityID:   l.SourceEntityID,
			TargetEntityID:   l.TargetEntityID,
			PipelineRunID:    l.PipelineRunID,
			PipelineName:     l.PipelineName,
			CreatedAt:        l.CreatedAt,
		}
	}

	return result, nil
}

// FindRelations returns upstream and downstream resource references
// for a specific resource identified by appName + entity_id.
func (s *ResourceChainStore) FindRelations(ctx context.Context, appName, entityID string) (*schema.ResourceRelations, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	relations := &schema.ResourceRelations{
		App:        appName,
		EntityID:   entityID,
		Upstream:   []schema.ResourceRef{},
		Downstream: []schema.ResourceRef{},
	}

	downLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.SourceApp(appName),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find downstream: %w", err)
	}
	for _, l := range downLinks {
		relations.Downstream = append(relations.Downstream, schema.ResourceRef{
			App:          l.TargetApp,
			EntityID:     l.TargetEntityID,
			Capability:   l.TargetCapability,
			PipelineName: l.PipelineName,
		})
	}

	upLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.TargetApp(appName),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find upstream: %w", err)
	}
	for _, l := range upLinks {
		relations.Upstream = append(relations.Upstream, schema.ResourceRef{
			App:          l.SourceApp,
			EntityID:     l.SourceEntityID,
			Capability:   l.SourceCapability,
			PipelineName: l.PipelineName,
		})
	}

	return relations, nil
}

// FindNodeRelations returns upstream and downstream edges for a node identified
// by (appName, capability, entityID). Optional pipelineName filter and time window.
func (s *ResourceChainStore) FindNodeRelations(ctx context.Context, appName, capability, entityID string, pipelineName string, since time.Duration) ([]schema.ResourceEdge, []schema.ResourceEdge, error) {
	if s == nil || s.client == nil {
		return nil, nil, nil
	}

	base := func() *gen.ResourceLinkQuery {
		q := s.client.ResourceLink.Query()
		if pipelineName != "" {
			q = q.Where(resourcelink.PipelineName(pipelineName))
		}
		if since > 0 {
			q = q.Where(resourcelink.CreatedAtGT(time.Now().Add(-since)))
		}
		return q
	}

	// downstream: source = this node
	downLinks, err := base().
		Where(
			resourcelink.SourceApp(appName),
			resourcelink.SourceCapability(capability),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find downstream edges: %w", err)
	}

	// upstream: target = this node
	upLinks, err := base().
		Where(
			resourcelink.TargetApp(appName),
			resourcelink.TargetCapability(capability),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find upstream edges: %w", err)
	}

	toEdges := func(links []*gen.ResourceLink) []schema.ResourceEdge {
		edges := make([]schema.ResourceEdge, len(links))
		for i, l := range links {
			edges[i] = schema.ResourceEdge{
				SourceApp:        l.SourceApp,
				SourceCapability: l.SourceCapability,
				SourceEntityID:   l.SourceEntityID,
				TargetApp:        l.TargetApp,
				TargetCapability: l.TargetCapability,
				TargetEntityID:   l.TargetEntityID,
				PipelineName:     l.PipelineName,
				CreatedAt:        l.CreatedAt,
			}
		}
		return edges
	}

	return toEdges(upLinks), toEdges(downLinks), nil
}

// SearchNodes returns distinct (app, capability, entity_id) tuples from
// resource_links where source_entity_id, target_entity_id, source_app,
// target_app, source_capability, or target_capability contains the query.
// cursor is a decimal offset into the deduplicated result stream; empty starts at 0.
func (s *ResourceChainStore) SearchNodes(ctx context.Context, query string, limit int, cursor string) ([]schema.ResourceRef, string, error) {
	if s == nil || s.client == nil || query == "" {
		return nil, "", nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return nil, "", fmt.Errorf("search nodes: invalid cursor")
		}
		offset = n
	}

	// Fetch candidate links using Ent-safe case-insensitive predicates.
	links, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.Or(
				resourcelink.SourceEntityIDContainsFold(query),
				resourcelink.TargetEntityIDContainsFold(query),
				resourcelink.SourceAppContainsFold(query),
				resourcelink.TargetAppContainsFold(query),
				resourcelink.SourceCapabilityContainsFold(query),
				resourcelink.TargetCapabilityContainsFold(query),
			),
		).
		Order(resourcelink.ByCreatedAt(sql.OrderDesc())).
		Limit((offset + limit + 1) * 2). // over-fetch to allow in-memory dedup + cursor window
		All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("search nodes: %w", err)
	}

	// Deduplicate by (app, capability, entity_id) in Go memory.
	seen := make(map[string]bool)
	var results []schema.ResourceRef
	lowerQuery := strings.ToLower(query)

	for _, rl := range links {
		addSourceResult(rl, lowerQuery, seen, &results)
		addTargetResult(rl, lowerQuery, seen, &results)
	}

	if offset > len(results) {
		return nil, "", nil
	}
	window := results[offset:]
	nextCursor := ""
	if len(window) > limit {
		window = window[:limit]
		nextCursor = strconv.Itoa(offset + limit)
	}

	return window, nextCursor, nil
}

// addSourceResult adds the source side of a resource link to results
// if any of its fields match the query.
func addSourceResult(rl *gen.ResourceLink, lowerQuery string, seen map[string]bool, results *[]schema.ResourceRef) {
	if !matchesField(rl.SourceEntityID, rl.SourceApp, rl.SourceCapability, lowerQuery) {
		return
	}
	key := rl.SourceApp + "|" + rl.SourceCapability + "|" + rl.SourceEntityID
	if seen[key] {
		return
	}
	seen[key] = true
	*results = append(*results, schema.ResourceRef{
		App:        rl.SourceApp,
		Capability: rl.SourceCapability,
		EntityID:   rl.SourceEntityID,
	})
}

// addTargetResult adds the target side of a resource link to results
// if any of its fields match the query.
func addTargetResult(rl *gen.ResourceLink, lowerQuery string, seen map[string]bool, results *[]schema.ResourceRef) {
	if !matchesField(rl.TargetEntityID, rl.TargetApp, rl.TargetCapability, lowerQuery) {
		return
	}
	key := rl.TargetApp + "|" + rl.TargetCapability + "|" + rl.TargetEntityID
	if seen[key] {
		return
	}
	seen[key] = true
	*results = append(*results, schema.ResourceRef{
		App:        rl.TargetApp,
		Capability: rl.TargetCapability,
		EntityID:   rl.TargetEntityID,
	})
}

// matchesField returns true if any of the given fields contain the query (case-insensitive).
func matchesField(entityID, appName, capability, lowerQuery string) bool {
	return strings.Contains(strings.ToLower(entityID), lowerQuery) ||
		strings.Contains(strings.ToLower(appName), lowerQuery) ||
		strings.Contains(strings.ToLower(capability), lowerQuery)
}

// ParameterIsExpired checks whether the given access token parameter has expired.
func ParameterIsExpired(p gen.Parameter) bool {
	return p.ExpiredAt.Before(time.Now())
}

// ---------------------------------------------------------------------------
// ClipStore
// ---------------------------------------------------------------------------

// ClipStore persists shareable markdown clips keyed by short slugs.
type ClipStore struct {
	client *gen.Client
}

// NewClipStore creates a ClipStore with the given ent client.
func NewClipStore(client *gen.Client) *ClipStore {
	return &ClipStore{client: client}
}

// CreateClip inserts a new clip row.
func (s *ClipStore) CreateClip(ctx context.Context, slug, title, description, content, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.Clip.Create().
		SetSlug(slug).
		SetTitle(title).
		SetDescription(description).
		SetContent(content).
		SetCreatedBy(createdBy).
		Save(ctx)
	return err
}

// GetClipBySlug retrieves a clip by slug. Returns nil if not found.
func (s *ClipStore) GetClipBySlug(ctx context.Context, slug string) (*gen.Clip, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	row, err := s.client.Clip.Query().
		Where(clip.SlugEQ(slug)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// ListClips returns clips ordered by created_at descending.
// When limit <= 0, all clips are returned.
func (s *ClipStore) ListClips(ctx context.Context, limit int) ([]*gen.Clip, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	q := s.client.Clip.Query().Order(gen.Desc(clip.FieldCreatedAt))
	if limit > 0 {
		q = q.Limit(limit)
	}
	return q.All(ctx)
}

// ---------------------------------------------------------------------------
// PageDataStore
// ---------------------------------------------------------------------------

// PageDataStore persists shareable view page data keyed by opaque tokens.
type PageDataStore struct {
	client *gen.Client
}

// NewPageDataStore creates a PageDataStore with the given ent client.
func NewPageDataStore(client *gen.Client) *PageDataStore {
	return &PageDataStore{client: client}
}

// CreatePageData inserts a new page_data row.
func (s *PageDataStore) CreatePageData(ctx context.Context, token string, pageType string, title string, data types.KV, createdBy string, expiresAt *time.Time) error {
	if s == nil || s.client == nil {
		return nil
	}
	m := s.client.PageData.Create().
		SetToken(token).
		SetType(pageType).
		SetTitle(title).
		SetCreatedBy(createdBy)
	if len(data) > 0 {
		m.SetData(data)
	}
	if expiresAt != nil {
		m.SetExpiresAt(*expiresAt)
	}
	_, err := m.Save(ctx)
	return err
}

// GetPageDataByToken retrieves a page_data row by token. Returns nil if not found.
func (s *PageDataStore) GetPageDataByToken(ctx context.Context, token string) (*gen.PageData, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	pageData, err := s.client.PageData.Query().
		Where(pagedata.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pageData, nil
}

// DeletePageData removes a page_data row by token. Returns the number of deleted rows.
func (s *PageDataStore) DeletePageData(ctx context.Context, token string) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	affected, err := s.client.PageData.Delete().
		Where(pagedata.TokenEQ(token)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// DeleteExpiredPageData removes rows where expires_at < now(). Returns the number of deleted rows.
func (s *PageDataStore) DeleteExpiredPageData(ctx context.Context) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	affected, err := s.client.PageData.Delete().
		Where(pagedata.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}

// ---------------------------------------------------------------------------
// NotifyStore
// ---------------------------------------------------------------------------

// NotifyStore provides CRUD for notification delivery records.
type NotifyStore struct {
	client *gen.Client
}

// NewNotifyStore returns a NotifyStore backed by the given Ent client.
func NewNotifyStore(client *gen.Client) *NotifyStore {
	return &NotifyStore{client: client}
}

// ListNotifyRecordsOptions holds filters and pagination for listing notification records.
type ListNotifyRecordsOptions struct {
	Limit  int    // max 100, default 20
	Cursor string // opaque cursor: ID value as string
}

// Record inserts a notification delivery record and returns the new row ID.
func (s *NotifyStore) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg string, payload map[string]any) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	create := s.client.NotificationRecord.Create().
		SetUID(uid).
		SetChannel(channel).
		SetTemplateID(templateID).
		SetSummary(summary).
		SetStatus(notificationrecord.Status(status)).
		SetErrorMsg(errorMsg).
		SetCreatedAt(time.Now())
	if payload != nil {
		create = create.SetPayloadSnapshot(payload)
	}
	rec, err := create.Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("record notification: %w", err)
	}
	return rec.ID, nil
}

// ListRecords returns per-user notification records, cursor-paginated (newest first).
func (s *NotifyStore) ListRecords(ctx context.Context, uid string, opts ListNotifyRecordsOptions) ([]*gen.NotificationRecord, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Desc(notificationrecord.FieldID)).
		Limit(opts.Limit + 1)

	if opts.Cursor != "" {
		id, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil {
			q = q.Where(notificationrecord.IDLT(id))
		}
	}

	records, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list notification records: %w", err)
	}

	var nextCursor string
	if len(records) > opts.Limit {
		nextCursor = strconv.FormatInt(records[opts.Limit-1].ID, 10)
		records = records[:opts.Limit]
	}

	return records, nextCursor, nil
}

// GetRecord returns a single notification record by ID.
func (s *NotifyStore) GetRecord(ctx context.Context, id int64) (*gen.NotificationRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rec, err := s.client.NotificationRecord.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get notification record: %w", err)
	}
	return rec, nil
}

// DeleteOldest removes the oldest records for a user exceeding keepN.
func (s *NotifyStore) DeleteOldest(ctx context.Context, uid string, keepN int) error {
	if s == nil || s.client == nil {
		return nil
	}
	if keepN <= 0 {
		return nil
	}

	total, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("count records for cleanup: %w", err)
	}
	if total <= keepN {
		return nil
	}

	excess := total - keepN
	oldest, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Asc(notificationrecord.FieldCreatedAt)).
		Limit(excess).
		All(ctx)
	if err != nil {
		return fmt.Errorf("find oldest records: %w", err)
	}

	ids := make([]int64, len(oldest))
	for i, rec := range oldest {
		ids[i] = rec.ID
	}
	_, err = s.client.NotificationRecord.Delete().
		Where(notificationrecord.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete oldest records: %w", err)
	}
	return nil
}

// LLMUsageStore persists and aggregates LLM token usage records.
type LLMUsageStore struct {
	client *gen.Client
}

// NewLLMUsageStore returns a store backed by the given ent client.
func NewLLMUsageStore(client *gen.Client) *LLMUsageStore {
	return &LLMUsageStore{client: client}
}

// NewLLMUsageStoreFromDatabase returns a store using the global database client.
func NewLLMUsageStoreFromDatabase() *LLMUsageStore {
	if Database == nil {
		return nil
	}
	client, ok := Database.GetDB().(*gen.Client)
	if !ok || client == nil {
		return nil
	}
	return NewLLMUsageStore(client)
}

// RecordLLMUsage inserts one LLM usage row.
func (s *LLMUsageStore) RecordLLMUsage(ctx context.Context, record *types.LLMUsageRecordInput) error {
	if s == nil || s.client == nil {
		return errors.New("store: llm usage store unavailable")
	}
	if record == nil {
		return errors.New("store: nil llm usage record")
	}
	if strings.TrimSpace(record.UID) == "" {
		return errors.New("store: llm usage uid required")
	}
	source := strings.TrimSpace(record.Source)
	if source == "" {
		source = types.TokenUsageSourceAgent
	}
	source = types.NormalizeTokenUsageSource(source)
	_, err := s.client.LLMUsageRecord.Create().
		SetUID(record.UID).
		SetSessionID(record.SessionID).
		SetModel(record.Model).
		SetPromptTokens(record.PromptTokens).
		SetCompletionTokens(record.CompletionTokens).
		SetTotalTokens(record.TotalTokens).
		SetCacheRead(record.CacheRead).
		SetCacheWrite(record.CacheWrite).
		SetSource(source).
		SetCreatedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("store: record llm usage: %w", err)
	}
	return nil
}

// TokenUsageStats aggregates usage for charts filtered by user and time range.
func (s *LLMUsageStore) TokenUsageStats(ctx context.Context, uid string, since, until time.Time, groupBy string) (*types.TokenUsageStats, error) {
	if s == nil || s.client == nil {
		return emptyTokenUsageStats(groupBy, since, until), nil
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return emptyTokenUsageStats(groupBy, since, until), nil
	}

	rows, err := s.client.LLMUsageRecord.Query().
		Where(
			llmusagerecord.UIDEQ(uid),
			llmusagerecord.CreatedAtGTE(since),
			llmusagerecord.CreatedAtLTE(until),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list llm usage: %w", err)
	}

	stats := emptyTokenUsageStats(groupBy, since, until)
	for _, row := range rows {
		stats.Summary.PromptTokens += int64(row.PromptTokens)
		stats.Summary.CompletionTokens += int64(row.CompletionTokens)
		stats.Summary.TotalTokens += int64(row.TotalTokens)
	}

	dailyBuckets := buildTokenUsageDailyBuckets(rows, groupBy)
	stats.Series = buildTokenUsageSeries(dailyBuckets, stats.PeriodStart, stats.PeriodEnd, groupBy)
	return stats, nil
}

func emptyTokenUsageStats(groupBy string, since, until time.Time) *types.TokenUsageStats {
	startDay := startOfUTCDayStore(since)
	endDay := startOfUTCDayStore(until)
	return &types.TokenUsageStats{
		Summary:     types.TokenUsageSummary{},
		Series:      []types.TokenUsageSeries{},
		PeriodStart: startDay.Format("2006-01-02"),
		PeriodEnd:   endDay.Format("2006-01-02"),
		Today:       time.Now().UTC().Format("2006-01-02"),
		GroupBy:     groupBy,
	}
}

func startOfUTCDayStore(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func buildTokenUsageDailyBuckets(rows []*gen.LLMUsageRecord, groupBy string) map[string]map[string]int64 {
	buckets := make(map[string]map[string]int64)
	add := func(label, day string, amount int64) {
		if buckets[label] == nil {
			buckets[label] = make(map[string]int64)
		}
		buckets[label][day] += amount
	}

	for _, row := range rows {
		day := startOfUTCDayStore(row.CreatedAt).Format("2006-01-02")
		switch groupBy {
		case "usage_type":
			source := types.NormalizeTokenUsageSource(row.Source)
			add(source, day, int64(row.TotalTokens))
		default:
			label := strings.TrimSpace(row.Model)
			if label == "" {
				label = "unknown"
			}
			add(label, day, int64(row.TotalTokens))
		}
	}
	return buckets
}

func buildTokenUsageSeries(buckets map[string]map[string]int64, periodStart, periodEnd, groupBy string) []types.TokenUsageSeries {
	if groupBy == "usage_type" {
		for _, source := range types.TokenUsageSourceOrder {
			if buckets[source] == nil {
				buckets[source] = make(map[string]int64)
			}
		}
	}

	if len(buckets) == 0 {
		return []types.TokenUsageSeries{}
	}

	start, err := time.ParseInLocation("2006-01-02", periodStart, time.UTC)
	if err != nil {
		return []types.TokenUsageSeries{}
	}
	end, err := time.ParseInLocation("2006-01-02", periodEnd, time.UTC)
	if err != nil {
		return []types.TokenUsageSeries{}
	}

	labels := make([]string, 0, len(buckets))
	if groupBy == "usage_type" {
		labels = append(labels, types.TokenUsageSourceOrder...)
	} else {
		for label := range buckets {
			labels = append(labels, label)
		}
		slices.Sort(labels)
	}

	series := make([]types.TokenUsageSeries, 0, len(labels))
	for _, label := range labels {
		points := make([]types.TokenUsagePoint, 0)
		var cumulative int64
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			dayKey := day.Format("2006-01-02")
			daily := buckets[label][dayKey]
			cumulative += daily
			points = append(points, types.TokenUsagePoint{
				Date:       dayKey,
				Daily:      daily,
				Cumulative: cumulative,
			})
		}
		displayLabel := label
		if groupBy == "usage_type" {
			displayLabel = types.TokenUsageSourceLabel(label)
		}
		series = append(series, types.TokenUsageSeries{
			Label:  displayLabel,
			Points: points,
		})
	}
	return series
}

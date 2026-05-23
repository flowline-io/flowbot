// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/app"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventconsumption"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pollingstate"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowsteprun"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
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
			return errors.New("store: db adapter is not specified. Please set `store_config.use_adapter` in `flowbot.json`")
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
	// GetDB returns the underlying DB connection
	GetDB() any

	// User management

	// UserCreate creates user record
	UserCreate(ctx context.Context, user *model.User) error
	// UserGet returns record for a given user ID
	UserGet(ctx context.Context, uid types.Uid) (*model.User, error)
	// UserGetAll returns user records for a given list of user IDs
	UserGetAll(ctx context.Context, ids ...types.Uid) ([]*model.User, error)
	// FirstUser returns the first user
	FirstUser(ctx context.Context) (*model.User, error)
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

	GetUsers(ctx context.Context) ([]*model.User, error)
	GetUserById(ctx context.Context, id int64) (*model.User, error)
	GetUserByFlag(ctx context.Context, flag string) (*model.User, error)
	CreatePlatformUser(ctx context.Context, item *model.PlatformUser) (int64, error)
	GetPlatformUsersByUserId(ctx context.Context, userId int64) ([]*model.PlatformUser, error)
	GetPlatformUserByFlag(ctx context.Context, flag string) (*model.PlatformUser, error)
	UpdatePlatformUser(ctx context.Context, item *model.PlatformUser) error
	GetPlatformChannelByFlag(ctx context.Context, flag string) (*model.PlatformChannel, error)
	GetPlatformChannelsByPlatformIds(ctx context.Context, platformIds []int64) ([]*model.PlatformChannel, error)
	GetPlatformChannelsByChannelId(ctx context.Context, channelId int64) (*model.PlatformChannel, error)
	CreatePlatformChannel(ctx context.Context, item *model.PlatformChannel) (int64, error)
	CreatePlatformChannelUser(ctx context.Context, item *model.PlatformChannelUser) (int64, error)
	GetPlatformChannelUsersByUserFlag(ctx context.Context, userFlag string) ([]*model.PlatformChannelUser, error)
	GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*model.PlatformChannelUser, error)
	GetMessage(ctx context.Context, flag string) (*model.Message, error)
	GetMessageByPlatform(ctx context.Context, platformId int64, platformMsgId string) (*model.Message, error)
	GetMessagesBySession(ctx context.Context, session string) ([]*model.Message, error)
	CreateMessage(ctx context.Context, message model.Message) error

	GetBot(ctx context.Context, id int64) (*model.Bot, error)
	GetBotByName(ctx context.Context, name string) (*model.Bot, error)
	CreateBot(ctx context.Context, bot *model.Bot) (int64, error)
	UpdateBot(ctx context.Context, bot *model.Bot) error
	DeleteBot(ctx context.Context, name string) error
	GetBots(ctx context.Context) ([]*model.Bot, error)
	GetPlatform(ctx context.Context, id int64) (*model.Platform, error)
	GetPlatformByName(ctx context.Context, name string) (*model.Platform, error)
	GetPlatforms(ctx context.Context) ([]*model.Platform, error)
	CreatePlatform(ctx context.Context, platform *model.Platform) (int64, error)
	GetChannel(ctx context.Context, id int64) (*model.Channel, error)
	GetChannelByName(ctx context.Context, name string) (*model.Channel, error)
	CreateChannel(ctx context.Context, channel *model.Channel) (int64, error)
	UpdateChannel(ctx context.Context, channel *model.Channel) error
	DeleteChannel(ctx context.Context, name string) error
	GetChannels(ctx context.Context) ([]*model.Channel, error)

	DataSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error
	DataGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
	DataList(ctx context.Context, uid types.Uid, topic string, filter types.DataFilter) ([]*model.Data, error)
	DataDelete(ctx context.Context, uid types.Uid, topic, key string) error
	ConfigSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error
	ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
	ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]*model.Config, error)
	ConfigDelete(ctx context.Context, uid types.Uid, topic, key string) error
	OAuthSet(ctx context.Context, oauth model.OAuth) error
	OAuthGet(ctx context.Context, uid types.Uid, topic, t string) (model.OAuth, error)
	OAuthGetAvailable(ctx context.Context, t string) ([]model.OAuth, error)
	FormSet(ctx context.Context, formId string, form model.Form) error
	FormGet(ctx context.Context, formId string) (model.Form, error)
	PageSet(ctx context.Context, pageId string, page model.Page) error
	PageGet(ctx context.Context, pageId string) (model.Page, error)
	BehaviorSet(ctx context.Context, behavior model.Behavior) error
	BehaviorGet(ctx context.Context, uid types.Uid, flag string) (model.Behavior, error)
	BehaviorList(ctx context.Context, uid types.Uid) ([]*model.Behavior, error)
	BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error
	ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	ParameterGet(ctx context.Context, flag string) (model.Parameter, error)
	ParameterDelete(ctx context.Context, flag string) error
	CreateInstruct(ctx context.Context, instruct *model.Instruct) (int64, error)
	ListInstruct(ctx context.Context, uid types.Uid, isExpire bool, limit int) ([]*model.Instruct, error)
	UpdateInstruct(ctx context.Context, instruct *model.Instruct) error
	CreateCounter(ctx context.Context, counter *model.Counter) (int64, error)
	IncreaseCounter(ctx context.Context, id, amount int64) error
	DecreaseCounter(ctx context.Context, id, amount int64) error
	ListCounter(ctx context.Context, uid types.Uid, topic string) ([]*model.Counter, error)
	GetCounter(ctx context.Context, id int64) (model.Counter, error)
	GetCounterByFlag(ctx context.Context, uid types.Uid, topic, flag string) (model.Counter, error)

	GetAgents(ctx context.Context) ([]*model.Agent, error)
	GetAgentByHostid(ctx context.Context, uid types.Uid, topic, hostid string) (*model.Agent, error)
	CreateAgent(ctx context.Context, agent *model.Agent) (int64, error)
	UpdateAgentLastOnlineAt(ctx context.Context, uid types.Uid, topic, hostid string, lastOnlineAt time.Time) error
	UpdateAgentOnlineDuration(ctx context.Context, uid types.Uid, topic, hostid string, offlineTime time.Time) error
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
		details["request"] = entry.Request
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
		SetBackend(event.Backend).
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
		"backend":         event.Backend,
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

func (s *PipelineStore) UpsertDefinition(ctx context.Context, name, description string, enabled bool, trigger, steps model.JSON) error {
	if s == nil || s.client == nil {
		return nil
	}
	existing, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if !gen.IsNotFound(err) {
			return err
		}
		now := time.Now()
		_, err = s.client.PipelineDefinition.Create().
			SetName(name).
			SetDescription(description).
			SetEnabled(enabled).
			SetTrigger(map[string]any(trigger)).
			SetSteps(map[string]any(steps)).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		return err
	}
	_, err = s.client.PipelineDefinition.UpdateOneID(existing.ID).
		SetDescription(description).
		SetEnabled(enabled).
		SetTrigger(map[string]any(trigger)).
		SetSteps(map[string]any(steps)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) CreateRun(ctx context.Context, pipelineName, eventID, eventType string) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	run, err := s.client.PipelineRun.Create().
		SetPipelineName(pipelineName).
		SetEventID(eventID).
		SetEventType(eventType).
		SetStatus(int(model.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineRun{
		ID:             run.ID,
		PipelineName:   run.PipelineName,
		EventID:        run.EventID,
		EventType:      run.EventType,
		Status:         model.PipelineState(run.Status),
		Error:          run.Error,
		CheckpointData: model.JSON(run.CheckpointData),
		LastHeartbeat:  run.LastHeartbeat,
		StartedAt:      run.StartedAt,
		CompletedAt:    run.CompletedAt,
		CreatedAt:      run.CreatedAt,
	}, nil
}

func (s *PipelineStore) UpdateRunStatus(ctx context.Context, runID int64, status model.PipelineState, errMsg string) error {
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

func (s *PipelineStore) CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error) {
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
		SetStatus(int(model.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineStepRun{
		ID:            sr.ID,
		PipelineRunID: sr.PipelineRunID,
		StepName:      sr.StepName,
		Capability:    sr.Capability,
		Operation:     sr.Operation,
		Params:        model.JSON(sr.Params),
		Result:        model.JSON(sr.Result),
		Attempt:       sr.Attempt,
		RetryConfig:   model.JSON(sr.RetryConfig),
		Status:        model.PipelineState(sr.Status),
		Error:         sr.Error,
		StartedAt:     sr.StartedAt,
		CompletedAt:   sr.CompletedAt,
		CreatedAt:     sr.CreatedAt,
	}, nil
}

func (s *PipelineStore) UpdateStepRun(ctx context.Context, stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineStepRun.UpdateOneID(stepRunID).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == model.PipelineDone || status == model.PipelineCancel {
		now := time.Now()
		upd = upd.SetCompletedAt(now)
	}
	if result != nil {
		upd = upd.SetResult(map[string]any(result))
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
func (s *PipelineStore) GetIncompleteRuns(ctx context.Context) ([]*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Status(int(model.PipelineStart))).
		Order(pipelinerun.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*model.PipelineRun, len(runs))
	for i, r := range runs {
		result[i] = &model.PipelineRun{
			ID:             r.ID,
			PipelineName:   r.PipelineName,
			EventID:        r.EventID,
			EventType:      r.EventType,
			Status:         model.PipelineState(r.Status),
			Error:          r.Error,
			CheckpointData: model.JSON(r.CheckpointData),
			LastHeartbeat:  r.LastHeartbeat,
			StartedAt:      r.StartedAt,
			CompletedAt:    r.CompletedAt,
			CreatedAt:      r.CreatedAt,
		}
	}
	return result, nil
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
func (s *PipelineStore) GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineRun{
		ID:             run.ID,
		PipelineName:   run.PipelineName,
		EventID:        run.EventID,
		EventType:      run.EventType,
		Status:         model.PipelineState(run.Status),
		Error:          run.Error,
		CheckpointData: model.JSON(run.CheckpointData),
		LastHeartbeat:  run.LastHeartbeat,
		StartedAt:      run.StartedAt,
		CompletedAt:    run.CompletedAt,
		CreatedAt:      run.CreatedAt,
	}, nil
}

// RecordResourceLink inserts a resource link with UPSERT semantics.
func (s *PipelineStore) RecordResourceLink(ctx context.Context, link model.ResourceLink) error {
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
func (s *WorkflowRunStore) CreateRun(ctx context.Context, workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	wr, err := s.client.WorkflowRun.Create().
		SetWorkflowName(workflowName).
		SetWorkflowFile(workflowFile).
		SetStatus(int(model.WorkflowRunRunning)).
		SetTriggerType(triggerType).
		SetTriggerInfo(map[string]any(triggerInfo)).
		SetInputParams(map[string]any(inputParams)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowRunToModel(wr), nil
}

// UpdateRunStatus updates the status, error, and completed_at of a workflow run.
func (s *WorkflowRunStore) UpdateRunStatus(ctx context.Context, runID int64, status model.WorkflowRunState, errMsg string) error {
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
func (s *WorkflowRunStore) CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error) {
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
		SetStatus(int(model.WorkflowRunRunning)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowStepRunToModel(sr), nil
}

// UpdateStepRun updates the status, result, error, and attempt count of a workflow step run.
// completed_at is only set for terminal states (Done, Failed).
func (s *WorkflowRunStore) UpdateStepRun(ctx context.Context, stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	u := s.client.WorkflowStepRun.Update().
		Where(workflowsteprun.IDEQ(stepRunID)).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == model.WorkflowRunDone || status == model.WorkflowRunFailed {
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
	cp := model.JSON{}
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
func (s *WorkflowRunStore) GetIncompleteRuns(ctx context.Context) ([]*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.WorkflowRun.Query().
		Where(workflowrun.StatusEQ(int(model.WorkflowRunRunning))).
		Order(gen.Asc(workflowrun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*model.WorkflowRun, len(runs))
	for i, r := range runs {
		result[i] = genWorkflowRunToModel(r)
	}
	return result, nil
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
func (s *WorkflowRunStore) GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	wr, err := s.client.WorkflowRun.Query().
		Where(workflowrun.IDEQ(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return genWorkflowRunToModel(wr), nil
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

// genWorkflowRunToModel converts an Ent WorkflowRun entity to the model type.
func genWorkflowRunToModel(wr *gen.WorkflowRun) *model.WorkflowRun {
	return &model.WorkflowRun{
		ID:             wr.ID,
		WorkflowName:   wr.WorkflowName,
		WorkflowFile:   wr.WorkflowFile,
		Status:         model.WorkflowRunState(wr.Status),
		TriggerType:    wr.TriggerType,
		TriggerInfo:    model.JSON(wr.TriggerInfo),
		InputParams:    model.JSON(wr.InputParams),
		CheckpointData: model.JSON(wr.CheckpointData),
		LastHeartbeat:  wr.LastHeartbeat,
		Error:          wr.Error,
		StartedAt:      wr.StartedAt,
		CompletedAt:    wr.CompletedAt,
		CreatedAt:      wr.CreatedAt,
	}
}

// genWorkflowStepRunToModel converts an Ent WorkflowStepRun entity to the model type.
func genWorkflowStepRunToModel(sr *gen.WorkflowStepRun) *model.WorkflowStepRun {
	return &model.WorkflowStepRun{
		ID:            sr.ID,
		WorkflowRunID: sr.WorkflowRunID,
		StepID:        sr.StepID,
		StepName:      sr.StepName,
		Action:        sr.Action,
		ActionType:    sr.ActionType,
		Params:        model.JSON(sr.Params),
		Result:        model.JSON(sr.Result),
		Attempt:       sr.Attempt,
		Status:        model.WorkflowRunState(sr.Status),
		Error:         sr.Error,
		StartedAt:     sr.StartedAt,
		CompletedAt:   sr.CompletedAt,
		CreatedAt:     sr.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// HubStore
// ---------------------------------------------------------------------------

// HubStore persists homelab discovery data to the database.
type HubStore struct {
	client *gen.Client
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

func appJSON(ha homelab.App) (model.JSON, error) {
	raw, err := sonic.Marshal(ha)
	if err != nil {
		return nil, fmt.Errorf("marshal homelab app: %w", err)
	}
	var info model.JSON
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
func (s *ResourceChainStore) FindResourcesByTag(ctx context.Context, key, value string, limit int, cursor string) ([]*model.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tagJSON := fmt.Sprintf(`{"%s":"%s"}`, key, value)
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

	result := make([]*model.DataEvent, len(events))
	for i, e := range events {
		result[i] = &model.DataEvent{
			EventID:    e.EventID,
			EventType:  e.EventType,
			Source:     e.Source,
			Capability: e.Capability,
			Operation:  e.Operation,
			Backend:    e.Backend,
			App:        e.App,
			EntityID:   e.EntityID,
			CreatedAt:  e.CreatedAt,
		}
		if e.Data != nil {
			result[i].Data = model.JSON(e.Data)
		}
		if e.Tags != nil {
			result[i].Tags = model.JSON(e.Tags)
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
func (s *ResourceChainStore) FindResourceLinks(ctx context.Context, eventIDs []string) ([]*model.ResourceLink, error) {
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

	result := make([]*model.ResourceLink, len(links))
	for i, l := range links {
		result[i] = &model.ResourceLink{
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
func (s *ResourceChainStore) FindRelations(ctx context.Context, appName, entityID string) (*model.ResourceRelations, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	relations := &model.ResourceRelations{
		App:        appName,
		EntityID:   entityID,
		Upstream:   []model.ResourceRef{},
		Downstream: []model.ResourceRef{},
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
		relations.Downstream = append(relations.Downstream, model.ResourceRef{
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
		relations.Upstream = append(relations.Upstream, model.ResourceRef{
			App:          l.SourceApp,
			EntityID:     l.SourceEntityID,
			Capability:   l.SourceCapability,
			PipelineName: l.PipelineName,
		})
	}

	return relations, nil
}

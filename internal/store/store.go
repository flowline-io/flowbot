package store

import (
	"errors"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
)

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
		panic("store: Register adapter is nil")
	}

	adapterName := a.GetName()
	if _, ok := availableAdapters[adapterName]; ok {
		panic("store: adapter '" + adapterName + "' is already registered")
	}
	availableAdapters[adapterName] = a
}

// FS Media handler
var FS media.Handler

// Registered media/file handlers.
var fileHandlers map[string]media.Handler

// RegisterMediaHandler saves reference to a media handler (file upload-download handler).
func RegisterMediaHandler(name string, mh media.Handler) {
	if fileHandlers == nil {
		fileHandlers = make(map[string]media.Handler)
	}

	if mh == nil {
		panic("RegisterMediaHandler: handler is nil")
	}
	if _, dup := fileHandlers[name]; dup {
		panic("RegisterMediaHandler: called twice for handler " + name)
	}
	fileHandlers[name] = mh
}

// UseMediaHandler sets specified media handler as default.
func UseMediaHandler(name, config string) error {
	mediaHandler := fileHandlers[name]
	if mediaHandler == nil {
		panic("UseMediaHandler: unknown handler '" + name + "'")
	}
	FS = mediaHandler
	return mediaHandler.Init(config)
}

// PersistentStorageInterface defines methods used for interation with persistent storage.
type PersistentStorageInterface interface {
	Open(jsonConfig config.StoreType) error
	Close() error
	IsOpen() bool
	GetAdapter() Adapter
	DbStats() func() interface{}
}

// Store is the main object for interacting with persistent storage.
var Store PersistentStorageInterface

type storeObj struct{}

func (s storeObj) Open(jsonConfig config.StoreType) error {
	if err := openAdapter(jsonConfig); err != nil {
		return err
	}
	return nil
}

func (s storeObj) Close() error {
	if adp.IsOpen() {
		return adp.Close()
	}

	return nil
}

func (s storeObj) GetAdapter() Adapter {
	return adp
}

// IsOpen checks if persistent storage connection has been initialized.
func (storeObj) IsOpen() bool {
	if adp != nil {
		return adp.IsOpen()
	}

	return false
}

func (s storeObj) DbStats() func() interface{} {
	if !s.IsOpen() {
		return nil
	}
	return adp.Stats
}

type Adapter interface {
	// General

	// Open and configure the adapter
	Open(config config.StoreType) error
	// Close the adapter
	Close() error
	// IsOpen checks if the adapter is ready for use
	IsOpen() bool
	// GetName returns the name of the adapter
	GetName() string
	// Stats returns the DB connection stats object.
	Stats() interface{}

	// User management

	// UserCreate creates user record
	UserCreate(user *model.User) error
	// UserGet returns record for a given user ID
	UserGet(uid types.Uid) (*model.User, error)
	// UserGetAll returns user records for a given list of user IDs
	UserGetAll(ids ...types.Uid) ([]*model.User, error)
	// UserDelete deletes user record
	UserDelete(uid types.Uid, hard bool) error
	// UserUpdate updates user record
	UserUpdate(uid types.Uid, update types.KV) error

	// File upload records. The files are stored outside the database.

	// FileStartUpload initializes a file upload.
	FileStartUpload(fd *types.FileDef) error
	// FileFinishUpload marks file upload as completed, successfully or otherwise.
	FileFinishUpload(fd *types.FileDef, success bool, size int64) (*types.FileDef, error)
	// FileGet fetches a record of a specific file
	FileGet(fid string) (*types.FileDef, error)
	// FileDeleteUnused deletes records where UseCount is zero. If olderThan is non-zero, deletes
	// unused records with UpdatedAt before olderThan.
	// Returns array of FileDef.Location of deleted filerecords so actual files can be deleted too.
	FileDeleteUnused(olderThan time.Time, limit int) ([]string, error)

	GetUsers() ([]*model.User, error)
	GetUserById(id int64) (*model.User, error)
	GetUserByFlag(flag string) (*model.User, error)
	CreatePlatformUser(item *model.PlatformUser) (int64, error)
	GetPlatformUsersByUserId(userId int64) ([]*model.PlatformUser, error)
	GetPlatformUserByFlag(flag string) (*model.PlatformUser, error)
	UpdatePlatformUser(item *model.PlatformUser) error
	GetPlatformChannelByFlag(flag string) (*model.PlatformChannel, error)
	GetPlatformChannelsByPlatformIds(platformIds []int64) ([]*model.PlatformChannel, error)
	CreatePlatformChannel(item *model.PlatformChannel) (int64, error)
	GetMessage(flag string) (*model.Message, error)
	GetMessageByPlatform(platformId int64, platformMsgId string) (*model.Message, error)
	CreateMessage(message model.Message) error

	GetBot(id int64) (*model.Bot, error)
	GetBotByName(name string) (*model.Bot, error)
	CreateBot(bot *model.Bot) (int64, error)
	UpdateBot(bot *model.Bot) error
	DeleteBot(name string) error
	GetBots() ([]*model.Bot, error)
	GetPlatform(id int64) (*model.Platform, error)
	GetPlatformByName(name string) (*model.Platform, error)
	GetPlatforms() ([]*model.Platform, error)
	CreatePlatform(platform *model.Platform) (int64, error)
	GetChannel(id int64) (*model.Channel, error)
	GetChannelByName(name string) (*model.Channel, error)
	CreateChannel(channel *model.Channel) (int64, error)
	UpdateChannel(channel *model.Channel) error
	DeleteChannel(name string) error
	GetChannels() ([]*model.Channel, error)

	DataSet(uid types.Uid, topic, key string, value types.KV) error
	DataGet(uid types.Uid, topic, key string) (types.KV, error)
	DataList(uid types.Uid, topic string, filter types.DataFilter) ([]*model.Data, error)
	DataDelete(uid types.Uid, topic, key string) error
	ConfigSet(uid types.Uid, topic, key string, value types.KV) error
	ConfigGet(uid types.Uid, topic, key string) (types.KV, error)
	OAuthSet(oauth model.OAuth) error
	OAuthGet(uid types.Uid, topic, t string) (model.OAuth, error)
	OAuthGetAvailable(t string) ([]model.OAuth, error)
	FormSet(formId string, form model.Form) error
	FormGet(formId string) (model.Form, error)
	PageSet(pageId string, page model.Page) error
	PageGet(pageId string) (model.Page, error)
	BehaviorSet(behavior model.Behavior) error
	BehaviorGet(uid types.Uid, flag string) (model.Behavior, error)
	BehaviorList(uid types.Uid) ([]*model.Behavior, error)
	BehaviorIncrease(uid types.Uid, flag string, number int) error
	ParameterSet(flag string, params types.KV, expiredAt time.Time) error
	ParameterGet(flag string) (model.Parameter, error)
	ParameterDelete(flag string) error
	CreateInstruct(instruct *model.Instruct) (int64, error)
	ListInstruct(uid types.Uid, isExpire bool) ([]*model.Instruct, error)
	UpdateInstruct(instruct *model.Instruct) error
	ListWebhook(uid types.Uid) ([]*model.Webhook, error)
	CreateWebhook(webhook *model.Webhook) (int64, error)
	UpdateWebhook(webhook *model.Webhook) error
	DeleteWebhook(id int64) error
	IncreaseWebhookCount(id int64) error
	GetWebhookBySecret(secret string) (*model.Webhook, error)
	GetWebhookByUidAndFlag(uid types.Uid, flag string) (*model.Webhook, error)

	GetObjectiveByID(id int64) (*model.Objective, error)
	GetObjectiveBySequence(uid types.Uid, topic string, sequence int64) (*model.Objective, error)
	ListObjectives(uid types.Uid, topic string) ([]*model.Objective, error)
	CreateObjective(objective *model.Objective) (int64, error)
	UpdateObjective(objective *model.Objective) error
	DeleteObjective(id int64) error
	DeleteObjectiveBySequence(uid types.Uid, topic string, sequence int64) error
	GetKeyResultByID(id int64) (*model.KeyResult, error)
	GetKeyResultBySequence(uid types.Uid, topic string, sequence int64) (*model.KeyResult, error)
	ListKeyResults(uid types.Uid, topic string) ([]*model.KeyResult, error)
	ListKeyResultsById(id []int64) ([]*model.KeyResult, error)
	ListKeyResultsByObjectiveId(objectiveId int64) ([]*model.KeyResult, error)
	CreateKeyResult(keyResult *model.KeyResult) (int64, error)
	UpdateKeyResult(keyResult *model.KeyResult) error
	DeleteKeyResult(id int64) error
	DeleteKeyResultBySequence(uid types.Uid, topic string, sequence int64) error
	AggregateObjectiveValue(id int64) error
	AggregateKeyResultValue(id int64) error
	CreateKeyResultValue(keyResultValue *model.KeyResultValue) (int64, error)
	DeleteKeyResultValue(id int64) error
	GetKeyResultValues(keyResultId int64) ([]*model.KeyResultValue, error)
	GetKeyResultValue(id int64) (*model.KeyResultValue, error)
	CreateTodo(todo *model.Todo) (int64, error)
	ListTodos(uid types.Uid, topic string) ([]*model.Todo, error)
	ListRemindTodos(uid types.Uid, topic string) ([]*model.Todo, error)
	GetTodo(id int64) (*model.Todo, error)
	GetTodoBySequence(uid types.Uid, topic string, sequence int64) (*model.Todo, error)
	CompleteTodo(id int64) error
	CompleteTodoBySequence(uid types.Uid, topic string, sequence int64) error
	UpdateTodo(todo *model.Todo) error
	DeleteTodo(id int64) error
	DeleteTodoBySequence(uid types.Uid, topic string, sequence int64) error
	CreateReview(review *model.Review) (int64, error)
	UpdateReview(review *model.Review) error
	ListReviews(uid types.Uid, topic string) ([]*model.Review, error)
	GetReviewByID(id int64) (*model.Review, error)
	CreateReviewEvaluation(evaluation *model.ReviewEvaluation) (int64, error)
	UpdateReviewEvaluation(evaluation *model.ReviewEvaluation) error
	ListReviewEvaluations(uid types.Uid, topic string, reviewID int64) ([]*model.ReviewEvaluation, error)
	GetReviewEvaluationByID(id int64) (*model.ReviewEvaluation, error)
	CreateCycle(cycle *model.Cycle) (int64, error)
	UpdateCycle(cycle *model.Cycle) error
	ListCycles(uid types.Uid, topic string) ([]*model.Cycle, error)
	GetCycleByID(id int64) (*model.Cycle, error)

	CreateCounter(counter *model.Counter) (int64, error)
	IncreaseCounter(id, amount int64) error
	DecreaseCounter(id, amount int64) error
	ListCounter(uid types.Uid, topic string) ([]*model.Counter, error)
	GetCounter(id int64) (model.Counter, error)
	GetCounterByFlag(uid types.Uid, topic string, flag string) (model.Counter, error)

	CreateWorkflow(workflow *model.Workflow, script *model.WorkflowScript, dag *model.Dag, triggers []*model.WorkflowTrigger) (int64, error)
	GetWorkflow(id int64) (*model.Workflow, error)
	UpdateWorkflow(workflow *model.Workflow, script *model.WorkflowScript, dag *model.Dag, triggers []*model.WorkflowTrigger) error
	UpdateWorkflowState(id int64, state model.WorkflowState) error
	ListWorkflows(uid types.Uid, topic string) ([]*model.Workflow, error)
	IncreaseWorkflowCount(id int64, successful int32, failed int32, running int32, canceled int32) error
	IncreaseWorkflowTriggerCount(id int64, count int32) error
	DeleteWorkflow(id int64) error
	CreateWorkflowScript(item *model.WorkflowScript) (int64, error)
	GetWorkflowScript(id int64) (any, error)
	GetWorkflowScriptByWorkflowId(workflowId int64) (any, error)
	CreateWorkflowTrigger(item *model.WorkflowTrigger) (int64, error)
	UpdateWorkflowTrigger(item *model.WorkflowTrigger) error
	DeleteWorkflowTrigger(id int64) error
	ListWorkflowTriggerByType(t model.TriggerType) ([]*model.WorkflowTrigger, error)
	UpdateDag(item *model.Dag) error
	GetDag(id int64) (*model.Dag, error)
	GetJob(id int64) (*model.Job, error)
	CreateJob(item *model.Job) (int64, error)
	DeleteJob(id int64) error
	ListJobs(workflowID int64) ([]*model.Job, error)
	GetJobsByState(state model.JobState) ([]*model.Job, error)
	GetJobsByStates(states []model.JobState) ([]*model.Job, error)
	GetJobsByWorkflowId(workflowID int64) ([]*model.Job, error)
	UpdateJobState(id int64, state model.JobState) error
	UpdateStep(id int64, item *model.Step) error
	UpdateStepState(id int64, state model.StepState) error
	UpdateStepInput(id int64, input types.KV) error
	UpdateStepOutput(id int64, output types.KV) error
	CreateStep(step *model.Step) (int64, error)
	CreateSteps(steps []*model.Step) error
	GetStepsByState(state model.StepState) ([]*model.Step, error)
	GetStepsByDepend(jobId int64, depend []string) ([]*model.Step, error)
	GetStepsByJobId(jobId int64) ([]*model.Step, error)
}

var Database Adapter

func Init() {
	Store = storeObj{}
	Database = availableAdapters["mysql"] // default use mysql
}

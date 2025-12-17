package store

import (
	"errors"
	"fmt"
	"time"

	storeMigrate "github.com/flowline-io/flowbot/internal/store/migrate"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	_ "github.com/go-sql-driver/mysql" //revive:disable
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
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
		flog.Fatal("store: Register adapter is nil")
		return
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
	db, err := adp.GetDB().DB()
	if err != nil {
		return err
	}
	driver, _ := mysql.WithInstance(db, &mysql.Config{})

	d, err := iofs.New(storeMigrate.Fs, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", d, "mysql", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
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
	fmt.Printf("%s info %s media: handler '%s' registered\n", time.Now().Format(time.DateTime), utils.FileAndLine(), name)
}

// UseMediaHandler sets specified media handler as default.
func UseMediaHandler(name, config string) error {
	mediaHandler := fileHandlers[name]
	if mediaHandler == nil {
		flog.Fatal("UseMediaHandler: unknown handler %s", name)
		return fmt.Errorf("unknown handler %s", name)
	}
	FileSystem = mediaHandler
	return mediaHandler.Init(config)
}

// PersistentStorageInterface defines methods used for interaction with persistent storage.
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
	// GetDB returns the underlying DB connection
	GetDB() *gorm.DB

	// User management

	// UserCreate creates user record
	UserCreate(user *model.User) error
	// UserGet returns record for a given user ID
	UserGet(uid types.Uid) (*model.User, error)
	// UserGetAll returns user records for a given list of user IDs
	UserGetAll(ids ...types.Uid) ([]*model.User, error)
	// FirstUser returns the first user
	FirstUser() (*model.User, error)
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
	GetPlatformChannelsByChannelId(channelId int64) (*model.PlatformChannel, error)
	CreatePlatformChannel(item *model.PlatformChannel) (int64, error)
	CreatePlatformChannelUser(item *model.PlatformChannelUser) (int64, error)
	GetPlatformChannelUsersByUserFlag(userFlag string) ([]*model.PlatformChannelUser, error)
	GetMessage(flag string) (*model.Message, error)
	GetMessageByPlatform(platformId int64, platformMsgId string) (*model.Message, error)
	GetMessagesBySession(session string) ([]*model.Message, error)
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
	ListConfigByPrefix(uid types.Uid, topic string, prefix string) ([]*model.Config, error)
	ConfigDelete(uid types.Uid, topic string, key string) error
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
	ListInstruct(uid types.Uid, isExpire bool, limit int) ([]*model.Instruct, error)
	UpdateInstruct(instruct *model.Instruct) error
	ListWebhook(uid types.Uid) ([]*model.Webhook, error)
	CreateWebhook(webhook *model.Webhook) (int64, error)
	UpdateWebhook(webhook *model.Webhook) error
	DeleteWebhook(id int64) error
	IncreaseWebhookCount(id int64) error
	GetWebhookBySecret(secret string) (*model.Webhook, error)
	GetWebhookByUidAndFlag(uid types.Uid, flag string) (*model.Webhook, error)

	CreateCounter(counter *model.Counter) (int64, error)
	IncreaseCounter(id, amount int64) error
	DecreaseCounter(id, amount int64) error
	ListCounter(uid types.Uid, topic string) ([]*model.Counter, error)
	GetCounter(id int64) (model.Counter, error)
	GetCounterByFlag(uid types.Uid, topic string, flag string) (model.Counter, error)

	GetAgents() ([]*model.Agent, error)
	GetAgentByHostid(uid types.Uid, topic string, hostid string) (*model.Agent, error)
	CreateAgent(agent *model.Agent) (int64, error)
	UpdateAgentLastOnlineAt(uid types.Uid, topic string, hostid string, lastOnlineAt time.Time) error
	UpdateAgentOnlineDuration(uid types.Uid, topic string, hostid string, offlineTime time.Time) error

	// Apps management
	GetApps() ([]*model.App, error)
	GetApp(id int64) (*model.App, error)
	GetAppByName(name string) (*model.App, error)
	CreateApp(app *model.App) (int64, error)
	UpdateApp(app *model.App) error
	DeleteApp(id int64) error

	// Flows management
	GetFlows(uid types.Uid, topic string) ([]*model.Flow, error)
	GetFlow(id int64) (*model.Flow, error)
	CreateFlow(flow *model.Flow) (int64, error)
	UpdateFlow(flow *model.Flow) error
	DeleteFlow(id int64) error

	// Flow nodes management
	GetFlowNodes(flowID int64) ([]*model.FlowNode, error)
	GetFlowNode(flowID int64, nodeID string) (*model.FlowNode, error)
	CreateFlowNode(node *model.FlowNode) (int64, error)
	UpdateFlowNode(node *model.FlowNode) error
	DeleteFlowNode(id int64) error
	DeleteFlowNodesByFlowID(flowID int64) error

	// Flow edges management
	GetFlowEdges(flowID int64) ([]*model.FlowEdge, error)
	CreateFlowEdge(edge *model.FlowEdge) (int64, error)
	UpdateFlowEdge(edge *model.FlowEdge) error
	DeleteFlowEdge(id int64) error
	DeleteFlowEdgesByFlowID(flowID int64) error

	// Executions management
	GetExecutions(flowID int64, limit int) ([]*model.Execution, error)
	GetExecution(executionID string) (*model.Execution, error)
	CreateExecution(execution *model.Execution) (int64, error)
	UpdateExecution(execution *model.Execution) error

	// Flow job (per-node execution) management
	CreateFlowJob(job *model.FlowJob) (int64, error)
	UpdateFlowJob(job *model.FlowJob) error
	GetFlowJobsByExecution(executionID string) ([]*model.FlowJob, error)

	// Connections management
	GetConnections(uid types.Uid, topic string) ([]*model.Connection, error)
	GetConnection(id int64) (*model.Connection, error)
	CreateConnection(conn *model.Connection) (int64, error)
	UpdateConnection(conn *model.Connection) error
	DeleteConnection(id int64) error

	// Authentications management
	GetAuthentications(uid types.Uid, topic string) ([]*model.Authentication, error)
	GetAuthentication(id int64) (*model.Authentication, error)
	CreateAuthentication(auth *model.Authentication) (int64, error)
	UpdateAuthentication(auth *model.Authentication) error
	DeleteAuthentication(id int64) error

	// Rate limits management
	GetRateLimits(flowID *int64, nodeID string) ([]*model.RateLimit, error)
	CreateRateLimit(limit *model.RateLimit) (int64, error)
	UpdateRateLimit(limit *model.RateLimit) error
	DeleteRateLimit(id int64) error
}

var Database Adapter

func Init() {
	Store = storeObj{}
	Database = availableAdapters["mysql"] // default use mysql
}

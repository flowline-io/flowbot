package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
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
	ListWebhook(ctx context.Context, uid types.Uid) ([]*model.Webhook, error)
	CreateWebhook(ctx context.Context, webhook *model.Webhook) (int64, error)
	UpdateWebhook(ctx context.Context, webhook *model.Webhook) error
	DeleteWebhook(ctx context.Context, id int64) error
	IncreaseWebhookCount(ctx context.Context, id int64) error
	GetWebhookBySecret(ctx context.Context, secret string) (*model.Webhook, error)
	GetWebhookByUidAndFlag(ctx context.Context, uid types.Uid, flag string) (*model.Webhook, error)

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
	Database = availableAdapters["postgres"] // default use postgres
}

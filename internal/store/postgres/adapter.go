package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	_ "github.com/jackc/pgx/v5/stdlib" //revive:disable:blank-imports pgx driver registration

	entsql "entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/behavior"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/bot"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/channel"
	configdata "github.com/flowline-io/flowbot/internal/store/ent/gen/configdata"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/counter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/data"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/fileupload"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/form"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/instruct"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/message"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/oauth"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/page"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platform"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchanneluser"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformuser"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/user"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/webhook"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	adapterName = "postgres"

	defaultDatabase = "flowbot"

	defaultMaxResults        = 1024
	defaultMaxMessageResults = 100

	txTimeoutMultiplier = 1.5
)

type configType struct {
	DSN                 string `json:"dsn,omitempty"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime     int    `json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime     int    `json:"conn_max_idle_time,omitempty"`
	SqlTimeout          int    `json:"sql_timeout,omitempty"`
	HealthCheckInterval int    `json:"pool_health_check_interval,omitempty"`
	HealthCheckTimeout  int    `json:"pool_health_check_timeout,omitempty"`
}

// Init registers the postgres adapter with the store layer.
func Init() {
	store.RegisterAdapter(&adapter{})
}

type adapter struct {
	client  *gen.Client
	db      *sql.DB
	poolMgr *PoolManager

	dbName            string
	maxResults        int
	maxMessageResults int
	sqlTimeout        time.Duration
	txTimeout         time.Duration
	open              bool
}

func (a *adapter) Open(jsonConfig config.StoreType) error {
	var conf configType
	if c, ok := jsonConfig.Adapters[adapterName]; ok {
		raw, err := sonic.Marshal(c)
		if err != nil {
			return fmt.Errorf("postgres: marshal adapter config: %w", err)
		}
		if err := sonic.Unmarshal(raw, &conf); err != nil {
			return fmt.Errorf("postgres: unmarshal adapter config: %w", err)
		}
	}

	if conf.DSN == "" {
		return errors.New("postgres: DSN is required")
	}

	if conf.SqlTimeout <= 0 {
		conf.SqlTimeout = 10
	}

	db, err := sql.Open("pgx", conf.DSN)
	if err != nil {
		return fmt.Errorf("postgres: open db: %w", err)
	}

	poolCfg := PoolConfig{
		MaxOpenConns:        conf.MaxOpenConns,
		MaxIdleConns:        conf.MaxIdleConns,
		ConnMaxLifetime:     conf.ConnMaxLifetime,
		ConnMaxIdleTime:     conf.ConnMaxIdleTime,
		HealthCheckInterval: conf.HealthCheckInterval,
		HealthCheckTimeout:  conf.HealthCheckTimeout,
	}
	poolMgr := NewPoolManager(db, poolCfg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.SqlTimeout)*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		poolMgr.Stop()
		_ = db.Close()
		return fmt.Errorf("postgres: ping db: %w", err)
	}

	drv := entsql.OpenDB("postgres", db)
	a.client = gen.NewClient(gen.Driver(drv))

	a.db = db
	a.poolMgr = poolMgr
	a.dbName = defaultDatabase
	a.maxResults = jsonConfig.MaxResults
	if a.maxResults <= 0 {
		a.maxResults = defaultMaxResults
	}
	a.maxMessageResults = defaultMaxMessageResults
	a.sqlTimeout = time.Duration(conf.SqlTimeout) * time.Second
	a.txTimeout = time.Duration(float64(conf.SqlTimeout)*txTimeoutMultiplier) * time.Second
	a.open = true

	poolMgr.Start(context.Background())
	flog.Info("postgres: adapter opened with database '%s'", a.dbName)
	return nil
}

func (a *adapter) Close() error {
	if a.poolMgr != nil {
		a.poolMgr.Stop()
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("postgres: close db: %w", err)
		}
	}
	a.open = false
	return nil
}

func (a *adapter) IsOpen() bool {
	return a.open
}

func (a *adapter) GetName() string {
	return adapterName
}

func (a *adapter) Stats() any {
	if a.db != nil {
		return a.db.Stats()
	}
	return nil
}

func (a *adapter) GetClient() *gen.Client {
	return a.client
}

func (a *adapter) GetDB() any {
	return a.client
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

func (a *adapter) UserCreate(usr *model.User) error {
	ctx := context.Background()
	_, err := a.client.User.Create().
		SetFlag(usr.Flag).
		SetName(usr.Name).
		SetTags(usr.Tags).
		SetState(int(usr.State)).
		SetCreatedAt(usr.CreatedAt).
		SetUpdatedAt(usr.UpdatedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create user: %w", err)
	}
	return nil
}

func (a *adapter) UserGet(uid types.Uid) (*model.User, error) {
	ctx := context.Background()
	u, err := a.client.User.Query().Where(user.FlagEQ(uid.String())).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user: %w", err)
	}
	return entUserToModel(u), nil
}

func (a *adapter) UserGetAll(ids ...types.Uid) ([]*model.User, error) {
	ctx := context.Background()
	q := a.client.User.Query()
	if len(ids) > 0 {
		flags := make([]string, len(ids))
		for i, id := range ids {
			flags[i] = id.String()
		}
		q = q.Where(user.FlagIn(flags...))
	}
	q = q.Limit(a.maxResults)
	users, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get all users: %w", err)
	}
	result := make([]*model.User, len(users))
	for i, u := range users {
		result[i] = entUserToModel(u)
	}
	return result, nil
}

func (a *adapter) FirstUser() (*model.User, error) {
	ctx := context.Background()
	u, err := a.client.User.Query().Order(gen.Asc(user.FieldID)).First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: first user: %w", err)
	}
	return entUserToModel(u), nil
}

func (a *adapter) UserDelete(uid types.Uid, hard bool) error {
	ctx := context.Background()
	if hard {
		_, err := a.client.User.Delete().Where(user.FlagEQ(uid.String())).Exec(ctx)
		if err != nil {
			return fmt.Errorf("postgres: hard delete user: %w", err)
		}
		return nil
	}
	_, err := a.client.User.Update().Where(user.FlagEQ(uid.String())).
		SetState(int(model.UserInactive)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: soft delete user: %w", err)
	}
	return nil
}

func (a *adapter) UserUpdate(uid types.Uid, update types.KV) error {
	ctx := context.Background()
	u := a.client.User.Update().Where(user.FlagEQ(uid.String()))
	if v, ok := update.String("name"); ok {
		u = u.SetName(v)
	}
	if v, ok := update.String("tags"); ok {
		u = u.SetTags(v)
	}
	if v, ok := update.Int64("state"); ok {
		u = u.SetState(int(v))
	}
	u = u.SetUpdatedAt(time.Now())
	_, err := u.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update user: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// File
// ---------------------------------------------------------------------------

func (a *adapter) FileStartUpload(fd *types.FileDef) error {
	ctx := context.Background()
	_, err := a.client.Fileupload.Create().
		SetUID(fd.User).
		SetFid(fd.Id).
		SetName(fd.Name).
		SetMimetype(fd.MimeType).
		SetSize(fd.Size).
		SetLocation(fd.Location).
		SetState(int(model.FileStart)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: file start upload: %w", err)
	}
	return nil
}

func (a *adapter) FileFinishUpload(fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	ctx := context.Background()
	st := int(model.FileFailed)
	if success {
		st = int(model.FileFinish)
	}
	_, err := a.client.Fileupload.Update().
		Where(fileupload.FidEQ(fd.Id)).
		SetSize(size).
		SetState(st).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: file finish upload: %w", err)
	}
	return a.FileGet(fd.Id)
}

func (a *adapter) FileGet(fid string) (*types.FileDef, error) {
	ctx := context.Background()
	u, err := a.client.Fileupload.Query().Where(fileupload.FidEQ(fid)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: file get: %w", err)
	}
	return entFileuploadToFileDef(u), nil
}

func (a *adapter) FileDeleteUnused(olderThan time.Time, limit int) ([]string, error) {
	ctx := context.Background()
	q := a.client.Fileupload.Query().
		Where(fileupload.StateEQ(int(model.FileFinish)))
	if !olderThan.IsZero() {
		q = q.Where(fileupload.UpdatedAtLT(olderThan))
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	files, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: file delete unused query: %w", err)
	}
	locations := make([]string, 0, len(files))
	for _, f := range files {
		locations = append(locations, f.Location)
	}
	if len(files) > 0 {
		ids := make([]int64, len(files))
		for i, f := range files {
			ids[i] = f.ID
		}
		_, err := a.client.Fileupload.Delete().Where(fileupload.IDIn(ids...)).Exec(ctx)
		if err != nil {
			return locations, fmt.Errorf("postgres: file delete unused: %w", err)
		}
	}
	return locations, nil
}

// ---------------------------------------------------------------------------
// Users (by id/flag)
// ---------------------------------------------------------------------------

func (a *adapter) GetUsers() ([]*model.User, error) {
	return a.UserGetAll()
}

func (a *adapter) GetUserById(id int64) (*model.User, error) {
	ctx := context.Background()
	u, err := a.client.User.Query().Where(user.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user by id: %w", err)
	}
	return entUserToModel(u), nil
}

func (a *adapter) GetUserByFlag(flag string) (*model.User, error) {
	return a.UserGet(types.Uid(flag))
}

// ---------------------------------------------------------------------------
// PlatformUser
// ---------------------------------------------------------------------------

func (a *adapter) CreatePlatformUser(item *model.PlatformUser) (int64, error) {
	ctx := context.Background()
	u, err := a.client.PlatformUser.Create().
		SetPlatformID(item.PlatformID).
		SetUserID(item.UserID).
		SetFlag(item.Flag).
		SetName(item.Name).
		SetEmail(item.Email).
		SetAvatarURL(item.AvatarURL).
		SetIsBot(item.IsBot).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform user: %w", err)
	}
	return u.ID, nil
}

func (a *adapter) GetPlatformUsersByUserId(userId int64) ([]*model.PlatformUser, error) {
	ctx := context.Background()
	users, err := a.client.PlatformUser.Query().Where(platformuser.UserIDEQ(userId)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform users by user id: %w", err)
	}
	result := make([]*model.PlatformUser, len(users))
	for i, u := range users {
		result[i] = entPlatformUserToModel(u)
	}
	return result, nil
}

func (a *adapter) GetPlatformUserByFlag(flag string) (*model.PlatformUser, error) {
	ctx := context.Background()
	u, err := a.client.PlatformUser.Query().Where(platformuser.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform user by flag: %w", err)
	}
	return entPlatformUserToModel(u), nil
}

func (a *adapter) UpdatePlatformUser(item *model.PlatformUser) error {
	ctx := context.Background()
	_, err := a.client.PlatformUser.Update().Where(platformuser.IDEQ(item.ID)).
		SetPlatformID(item.PlatformID).
		SetUserID(item.UserID).
		SetFlag(item.Flag).
		SetName(item.Name).
		SetEmail(item.Email).
		SetAvatarURL(item.AvatarURL).
		SetIsBot(item.IsBot).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: update platform user: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PlatformChannel
// ---------------------------------------------------------------------------

func (a *adapter) GetPlatformChannelByFlag(flag string) (*model.PlatformChannel, error) {
	ctx := context.Background()
	u, err := a.client.PlatformChannel.Query().Where(platformchannel.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by flag: %w", err)
	}
	return entPlatformChannelToModel(u), nil
}

func (a *adapter) GetPlatformChannelsByPlatformIds(platformIds []int64) ([]*model.PlatformChannel, error) {
	ctx := context.Background()
	channels, err := a.client.PlatformChannel.Query().Where(platformchannel.PlatformIDIn(platformIds...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channels by platform ids: %w", err)
	}
	result := make([]*model.PlatformChannel, len(channels))
	for i, c := range channels {
		result[i] = entPlatformChannelToModel(c)
	}
	return result, nil
}

func (a *adapter) GetPlatformChannelsByChannelId(channelId int64) (*model.PlatformChannel, error) {
	ctx := context.Background()
	u, err := a.client.PlatformChannel.Query().Where(platformchannel.ChannelIDEQ(channelId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by channel id: %w", err)
	}
	return entPlatformChannelToModel(u), nil
}

func (a *adapter) CreatePlatformChannel(item *model.PlatformChannel) (int64, error) {
	ctx := context.Background()
	u, err := a.client.PlatformChannel.Create().
		SetPlatformID(item.PlatformID).
		SetChannelID(item.ChannelID).
		SetFlag(item.Flag).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform channel: %w", err)
	}
	return u.ID, nil
}

// ---------------------------------------------------------------------------
// PlatformChannelUser
// ---------------------------------------------------------------------------

func (a *adapter) CreatePlatformChannelUser(item *model.PlatformChannelUser) (int64, error) {
	ctx := context.Background()
	u, err := a.client.PlatformChannelUser.Create().
		SetPlatformID(item.PlatformID).
		SetChannelFlag(item.ChannelFlag).
		SetUserFlag(item.UserFlag).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform channel user: %w", err)
	}
	return u.ID, nil
}

func (a *adapter) GetPlatformChannelUsersByUserFlag(userFlag string) ([]*model.PlatformChannelUser, error) {
	ctx := context.Background()
	users, err := a.client.PlatformChannelUser.Query().
		Where(platformchanneluser.UserFlagEQ(userFlag)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channel users by user flag: %w", err)
	}
	result := make([]*model.PlatformChannelUser, len(users))
	for i, u := range users {
		result[i] = entPlatformChannelUserToModel(u)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

func (a *adapter) GetMessage(flag string) (*model.Message, error) {
	ctx := context.Background()
	m, err := a.client.Message.Query().Where(message.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message: %w", err)
	}
	return entMessageToModel(m), nil
}

func (a *adapter) GetMessageByPlatform(platformId int64, platformMsgId string) (*model.Message, error) {
	ctx := context.Background()
	m, err := a.client.Message.Query().
		Where(message.PlatformIDEQ(platformId), message.PlatformMsgIDEQ(platformMsgId)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message by platform: %w", err)
	}
	return entMessageToModel(m), nil
}

func (a *adapter) GetMessagesBySession(session string) ([]*model.Message, error) {
	ctx := context.Background()
	messages, err := a.client.Message.Query().
		Where(message.SessionEQ(session)).
		Order(gen.Asc(message.FieldCreatedAt)).
		Limit(a.maxMessageResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get messages by session: %w", err)
	}
	result := make([]*model.Message, len(messages))
	for i, m := range messages {
		result[i] = entMessageToModel(m)
	}
	return result, nil
}

func (a *adapter) CreateMessage(msg model.Message) error {
	ctx := context.Background()
	c := a.client.Message.Create().
		SetFlag(msg.Flag).
		SetPlatformID(msg.PlatformID).
		SetPlatformMsgID(msg.PlatformMsgID).
		SetTopic(msg.Topic).
		SetRole(msg.Role).
		SetSession(msg.Session).
		SetState(int(msg.State)).
		SetCreatedAt(msg.CreatedAt).
		SetUpdatedAt(msg.UpdatedAt)
	if msg.Content != nil {
		c = c.SetContent(map[string]any(msg.Content))
	}
	if msg.DeletedAt != nil {
		c = c.SetDeletedAt(*msg.DeletedAt)
	}
	_, err := c.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create message: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Data (KV store)
// ---------------------------------------------------------------------------

func (a *adapter) DataSet(uid types.Uid, topic, key string, value types.KV) error {
	ctx := context.Background()
	existing, err := a.client.Data.Query().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: dataset query: %w", err)
	}

	if existing != nil {
		_, err = a.client.Data.Update().Where(data.IDEQ(existing.ID)).
			SetValue(map[string]any(value)).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = a.client.Data.Create().
			SetUID(uid.String()).
			SetTopic(topic).
			SetKey(key).
			SetValue(map[string]any(value)).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: dataset save: %w", err)
	}
	return nil
}

func (a *adapter) DataGet(uid types.Uid, topic, key string) (types.KV, error) {
	ctx := context.Background()
	d, err := a.client.Data.Query().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: dataget: %w", err)
	}
	return types.KV(d.Value), nil
}

func (a *adapter) DataList(uid types.Uid, topic string, filter types.DataFilter) ([]*model.Data, error) {
	ctx := context.Background()
	q := a.client.Data.Query().Where(data.UID(uid.String()), data.Topic(topic))
	if filter.Prefix != nil && *filter.Prefix != "" {
		q = q.Where(data.KeyHasPrefix(*filter.Prefix))
	}
	if filter.CreatedStart != nil {
		q = q.Where(data.CreatedAtGTE(*filter.CreatedStart))
	}
	if filter.CreatedEnd != nil {
		q = q.Where(data.CreatedAtLTE(*filter.CreatedEnd))
	}
	q = q.Order(gen.Asc(data.FieldCreatedAt)).Limit(a.maxResults)
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: datalist: %w", err)
	}
	result := make([]*model.Data, len(items))
	for i, d := range items {
		result[i] = &model.Data{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     model.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) DataDelete(uid types.Uid, topic string, key string) error {
	ctx := context.Background()
	_, err := a.client.Data.Delete().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: datadelete: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Config (KV config)
// ---------------------------------------------------------------------------

func (a *adapter) ConfigSet(uid types.Uid, topic, key string, value types.KV) error {
	ctx := context.Background()
	existing, err := a.client.ConfigData.Query().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: configset query: %w", err)
	}

	if existing != nil {
		_, err = a.client.ConfigData.Update().Where(configdata.IDEQ(existing.ID)).
			SetValue(map[string]any(value)).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = a.client.ConfigData.Create().
			SetUID(uid.String()).
			SetTopic(topic).
			SetKey(key).
			SetValue(map[string]any(value)).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: configset save: %w", err)
	}
	return nil
}

func (a *adapter) ConfigGet(uid types.Uid, topic, key string) (types.KV, error) {
	ctx := context.Background()
	d, err := a.client.ConfigData.Query().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: configget: %w", err)
	}
	return types.KV(d.Value), nil
}

func (a *adapter) ListConfigByPrefix(uid types.Uid, topic string, prefix string) ([]*model.Config, error) {
	ctx := context.Background()
	q := a.client.ConfigData.Query().Where(configdata.UID(uid.String()), configdata.Topic(topic))
	if prefix != "" {
		q = q.Where(configdata.KeyHasPrefix(prefix))
	}
	q = q.Order(gen.Asc(configdata.FieldCreatedAt)).Limit(a.maxResults)
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigbyprefix: %w", err)
	}
	result := make([]*model.Config, len(items))
	for i, d := range items {
		result[i] = &model.Config{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     model.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) ConfigDelete(uid types.Uid, topic string, key string) error {
	ctx := context.Background()
	_, err := a.client.ConfigData.Delete().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: configdelete: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// OAuth
// ---------------------------------------------------------------------------

func (a *adapter) OAuthSet(oauthModel model.OAuth) error {
	ctx := context.Background()
	existing, err := a.client.OAuth.Query().
		Where(
			oauth.UID(oauthModel.UID),
			oauth.Topic(oauthModel.Topic),
			oauth.Type(oauthModel.Type),
		).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: oauthset query: %w", err)
	}

	if existing != nil {
		u := a.client.OAuth.Update().Where(oauth.IDEQ(existing.ID)).
			SetName(oauthModel.Name).
			SetToken(oauthModel.Token).
			SetUpdatedAt(time.Now())
		if oauthModel.Extra != nil {
			u = u.SetExtra(map[string]any(oauthModel.Extra))
		}
		_, err = u.Save(ctx)
	} else {
		c := a.client.OAuth.Create().
			SetUID(oauthModel.UID).
			SetTopic(oauthModel.Topic).
			SetName(oauthModel.Name).
			SetType(oauthModel.Type).
			SetToken(oauthModel.Token).
			SetCreatedAt(oauthModel.CreatedAt).
			SetUpdatedAt(oauthModel.UpdatedAt)
		if oauthModel.Extra != nil {
			c = c.SetExtra(map[string]any(oauthModel.Extra))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: oauthset save: %w", err)
	}
	return nil
}

func (a *adapter) OAuthGet(uid types.Uid, topic, t string) (model.OAuth, error) {
	ctx := context.Background()
	o, err := a.client.OAuth.Query().
		Where(oauth.UID(uid.String()), oauth.Topic(topic), oauth.Type(t)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.OAuth{}, types.ErrNotFound
		}
		return model.OAuth{}, fmt.Errorf("postgres: oauthget: %w", err)
	}
	return entOAuthToModel(o), nil
}

func (a *adapter) OAuthGetAvailable(t string) ([]model.OAuth, error) {
	ctx := context.Background()
	oauths, err := a.client.OAuth.Query().Where(oauth.Type(t)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: oauthgetavailable: %w", err)
	}
	result := make([]model.OAuth, len(oauths))
	for i, o := range oauths {
		result[i] = entOAuthToModel(o)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

func (a *adapter) FormSet(formId string, formModel model.Form) error {
	ctx := context.Background()
	existing, err := a.client.Form.Query().Where(form.FormIDEQ(formId)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: formset query: %w", err)
	}

	if existing != nil {
		u := a.client.Form.Update().Where(form.IDEQ(existing.ID)).
			SetUID(formModel.UID).
			SetTopic(formModel.Topic).
			SetState(int(formModel.State)).
			SetUpdatedAt(time.Now())
		if formModel.Schema != nil {
			u = u.SetSchema(map[string]any(formModel.Schema))
		}
		if formModel.Values != nil {
			u = u.SetValues(map[string]any(formModel.Values))
		}
		if formModel.Extra != nil {
			u = u.SetExtra(map[string]any(formModel.Extra))
		}
		_, err = u.Save(ctx)
	} else {
		c := a.client.Form.Create().
			SetFormID(formModel.FormID).
			SetUID(formModel.UID).
			SetTopic(formModel.Topic).
			SetState(int(formModel.State)).
			SetCreatedAt(formModel.CreatedAt).
			SetUpdatedAt(formModel.UpdatedAt)
		if formModel.Schema != nil {
			c = c.SetSchema(map[string]any(formModel.Schema))
		}
		if formModel.Values != nil {
			c = c.SetValues(map[string]any(formModel.Values))
		}
		if formModel.Extra != nil {
			c = c.SetExtra(map[string]any(formModel.Extra))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: formset save: %w", err)
	}
	return nil
}

func (a *adapter) FormGet(formId string) (model.Form, error) {
	ctx := context.Background()
	f, err := a.client.Form.Query().Where(form.FormIDEQ(formId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Form{}, types.ErrNotFound
		}
		return model.Form{}, fmt.Errorf("postgres: formget: %w", err)
	}
	return entFormToModel(f), nil
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

func (a *adapter) PageSet(pageId string, pageModel model.Page) error {
	ctx := context.Background()
	existing, err := a.client.Page.Query().Where(page.PageIDEQ(pageId)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: pageset query: %w", err)
	}

	if existing != nil {
		u := a.client.Page.Update().Where(page.IDEQ(existing.ID)).
			SetUID(pageModel.UID).
			SetTopic(pageModel.Topic).
			SetType(string(pageModel.Type)).
			SetState(int(pageModel.State)).
			SetUpdatedAt(time.Now())
		if pageModel.Schema != nil {
			u = u.SetSchema(map[string]any(pageModel.Schema))
		}
		_, err = u.Save(ctx)
	} else {
		c := a.client.Page.Create().
			SetPageID(pageModel.PageID).
			SetUID(pageModel.UID).
			SetTopic(pageModel.Topic).
			SetType(string(pageModel.Type)).
			SetState(int(pageModel.State)).
			SetCreatedAt(pageModel.CreatedAt).
			SetUpdatedAt(pageModel.UpdatedAt)
		if pageModel.Schema != nil {
			c = c.SetSchema(map[string]any(pageModel.Schema))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: pageset save: %w", err)
	}
	return nil
}

func (a *adapter) PageGet(pageId string) (model.Page, error) {
	ctx := context.Background()
	p, err := a.client.Page.Query().Where(page.PageIDEQ(pageId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Page{}, types.ErrNotFound
		}
		return model.Page{}, fmt.Errorf("postgres: pageget: %w", err)
	}
	return entPageToModel(p), nil
}

// ---------------------------------------------------------------------------
// Behavior
// ---------------------------------------------------------------------------

func (a *adapter) BehaviorSet(behaviorModel model.Behavior) error {
	ctx := context.Background()
	existing, err := a.client.Behavior.Query().
		Where(behavior.UID(behaviorModel.UID), behavior.Flag(behaviorModel.Flag)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: behaviorset query: %w", err)
	}

	if existing != nil {
		u := a.client.Behavior.Update().Where(behavior.IDEQ(existing.ID)).
			SetCount(behaviorModel.Count_).
			SetUpdatedAt(time.Now())
		if behaviorModel.Extra != nil {
			u = u.SetExtra(map[string]any(*behaviorModel.Extra))
		}
		_, err = u.Save(ctx)
	} else {
		c := a.client.Behavior.Create().
			SetUID(behaviorModel.UID).
			SetFlag(behaviorModel.Flag).
			SetCount(behaviorModel.Count_).
			SetCreatedAt(behaviorModel.CreatedAt).
			SetUpdatedAt(behaviorModel.UpdatedAt)
		if behaviorModel.Extra != nil {
			c = c.SetExtra(map[string]any(*behaviorModel.Extra))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: behaviorset save: %w", err)
	}
	return nil
}

func (a *adapter) BehaviorGet(uid types.Uid, flag string) (model.Behavior, error) {
	ctx := context.Background()
	b, err := a.client.Behavior.Query().
		Where(behavior.UID(uid.String()), behavior.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Behavior{}, types.ErrNotFound
		}
		return model.Behavior{}, fmt.Errorf("postgres: behaviorget: %w", err)
	}
	return entBehaviorToModel(b), nil
}

func (a *adapter) BehaviorList(uid types.Uid) ([]*model.Behavior, error) {
	ctx := context.Background()
	behaviors, err := a.client.Behavior.Query().
		Where(behavior.UID(uid.String())).
		Order(gen.Asc(behavior.FieldCreatedAt)).
		Limit(a.maxResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: behaviorlist: %w", err)
	}
	result := make([]*model.Behavior, len(behaviors))
	for i, b := range behaviors {
		modelB := entBehaviorToModel(b)
		result[i] = &modelB
	}
	return result, nil
}

func (a *adapter) BehaviorIncrease(uid types.Uid, flag string, number int) error {
	ctx := context.Background()
	u := a.client.Behavior.Update().Where(behavior.UID(uid.String()), behavior.FlagEQ(flag))
	u = u.AddCount(int32(number)).SetUpdatedAt(time.Now())
	_, err := u.Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: behaviorincrease: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Parameter
// ---------------------------------------------------------------------------

func (a *adapter) ParameterSet(flag string, params types.KV, expiredAt time.Time) error {
	ctx := context.Background()
	existing, err := a.client.Parameter.Query().Where(parameter.FlagEQ(flag)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: parameterset query: %w", err)
	}

	if existing != nil {
		_, err = a.client.Parameter.Update().Where(parameter.IDEQ(existing.ID)).
			SetParams(map[string]any(params)).
			SetExpiredAt(expiredAt).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = a.client.Parameter.Create().
			SetFlag(flag).
			SetParams(map[string]any(params)).
			SetExpiredAt(expiredAt).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: parameterset save: %w", err)
	}
	return nil
}

func (a *adapter) ParameterGet(flag string) (model.Parameter, error) {
	ctx := context.Background()
	p, err := a.client.Parameter.Query().Where(parameter.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Parameter{}, types.ErrNotFound
		}
		return model.Parameter{}, fmt.Errorf("postgres: parameterget: %w", err)
	}
	return model.Parameter{
		ID:        p.ID,
		Flag:      p.Flag,
		Params:    model.JSON(p.Params),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		ExpiredAt: p.ExpiredAt,
	}, nil
}

func (a *adapter) ParameterDelete(flag string) error {
	ctx := context.Background()
	_, err := a.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: parameterdelete: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Instruct
// ---------------------------------------------------------------------------

func (a *adapter) CreateInstruct(instructModel *model.Instruct) (int64, error) {
	ctx := context.Background()
	c := a.client.Instruct.Create().
		SetNo(instructModel.No).
		SetUID(instructModel.UID).
		SetObject(string(instructModel.Object)).
		SetBot(instructModel.Bot).
		SetFlag(instructModel.Flag).
		SetPriority(int(instructModel.Priority)).
		SetState(int(instructModel.State)).
		SetExpireAt(instructModel.ExpireAt).
		SetCreatedAt(instructModel.CreatedAt).
		SetUpdatedAt(instructModel.UpdatedAt)
	if instructModel.Content != nil {
		c = c.SetContent(map[string]any(instructModel.Content))
	}
	u, err := c.Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createinstruct: %w", err)
	}
	return u.ID, nil
}

func (a *adapter) ListInstruct(uid types.Uid, isExpire bool, limit int) ([]*model.Instruct, error) {
	ctx := context.Background()
	q := a.client.Instruct.Query().Where(instruct.UID(uid.String()))
	if isExpire {
		q = q.Where(instruct.ExpireAtLTE(time.Now()))
	}
	q = q.Order(gen.Asc(instruct.FieldCreatedAt))
	if limit > 0 {
		q = q.Limit(limit)
	} else {
		q = q.Limit(a.maxResults)
	}
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listinstruct: %w", err)
	}
	result := make([]*model.Instruct, len(items))
	for i, ins := range items {
		result[i] = &model.Instruct{
			ID:        ins.ID,
			No:        ins.No,
			UID:       ins.UID,
			Object:    model.InstructObject(ins.Object),
			Bot:       ins.Bot,
			Flag:      ins.Flag,
			Content:   model.JSON(ins.Content),
			Priority:  model.InstructPriority(ins.Priority),
			State:     model.InstructState(ins.State),
			ExpireAt:  ins.ExpireAt,
			CreatedAt: ins.CreatedAt,
			UpdatedAt: ins.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) UpdateInstruct(instructModel *model.Instruct) error {
	ctx := context.Background()
	_, err := a.client.Instruct.Update().
		Where(instruct.NoEQ(instructModel.No)).
		SetState(int(instructModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: updateinstruct: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Webhook
// ---------------------------------------------------------------------------

func (a *adapter) ListWebhook(uid types.Uid) ([]*model.Webhook, error) {
	ctx := context.Background()
	webhooks, err := a.client.Webhook.Query().
		Where(webhook.UID(uid.String())).
		Order(gen.Asc(webhook.FieldCreatedAt)).
		Limit(a.maxResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listwebhook: %w", err)
	}
	result := make([]*model.Webhook, len(webhooks))
	for i, w := range webhooks {
		result[i] = entWebhookToModel(w)
	}
	return result, nil
}

func (a *adapter) CreateWebhook(webhookModel *model.Webhook) (int64, error) {
	ctx := context.Background()
	w, err := a.client.Webhook.Create().
		SetUID(webhookModel.UID).
		SetTopic(webhookModel.Topic).
		SetFlag(webhookModel.Flag).
		SetSecret(webhookModel.Secret).
		SetTriggerCount(webhookModel.TriggerCount).
		SetState(int(webhookModel.State)).
		SetCreatedAt(webhookModel.CreatedAt).
		SetUpdatedAt(webhookModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createwebhook: %w", err)
	}
	return w.ID, nil
}

func (a *adapter) UpdateWebhook(webhookModel *model.Webhook) error {
	ctx := context.Background()
	_, err := a.client.Webhook.Update().Where(webhook.IDEQ(webhookModel.ID)).
		SetUID(webhookModel.UID).
		SetTopic(webhookModel.Topic).
		SetFlag(webhookModel.Flag).
		SetSecret(webhookModel.Secret).
		SetTriggerCount(webhookModel.TriggerCount).
		SetState(int(webhookModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updatewebhook: %w", err)
	}
	return nil
}

func (a *adapter) DeleteWebhook(id int64) error {
	ctx := context.Background()
	_, err := a.client.Webhook.Delete().Where(webhook.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletewebhook: %w", err)
	}
	return nil
}

func (a *adapter) IncreaseWebhookCount(id int64) error {
	ctx := context.Background()
	_, err := a.client.Webhook.Update().Where(webhook.IDEQ(id)).
		AddTriggerCount(1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: increasewebhookcount: %w", err)
	}
	return nil
}

func (a *adapter) GetWebhookBySecret(secret string) (*model.Webhook, error) {
	ctx := context.Background()
	w, err := a.client.Webhook.Query().Where(webhook.SecretEQ(secret)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getwebhookbysecret: %w", err)
	}
	return entWebhookToModel(w), nil
}

func (a *adapter) GetWebhookByUidAndFlag(uid types.Uid, flag string) (*model.Webhook, error) {
	ctx := context.Background()
	w, err := a.client.Webhook.Query().
		Where(webhook.UID(uid.String()), webhook.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getwebhookbyuidandflag: %w", err)
	}
	return entWebhookToModel(w), nil
}

// ---------------------------------------------------------------------------
// Counter
// ---------------------------------------------------------------------------

func (a *adapter) CreateCounter(counterModel *model.Counter) (int64, error) {
	ctx := context.Background()
	c, err := a.client.Counter.Create().
		SetUID(counterModel.UID).
		SetTopic(counterModel.Topic).
		SetFlag(counterModel.Flag).
		SetDigit(counterModel.Digit).
		SetStatus(counterModel.Status).
		SetCreatedAt(counterModel.CreatedAt).
		SetUpdatedAt(counterModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createcounter: %w", err)
	}
	return c.ID, nil
}

func (a *adapter) IncreaseCounter(id, amount int64) error {
	ctx := context.Background()
	_, err := a.client.Counter.Update().Where(counter.IDEQ(id)).
		AddDigit(amount).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: increasecounter: %w", err)
	}
	return a.record(ctx, id, amount)
}

func (a *adapter) DecreaseCounter(id, amount int64) error {
	ctx := context.Background()
	_, err := a.client.Counter.Update().Where(counter.IDEQ(id)).
		AddDigit(-amount).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: decreasecounter: %w", err)
	}
	return a.record(ctx, id, -amount)
}

func (a *adapter) ListCounter(uid types.Uid, topic string) ([]*model.Counter, error) {
	ctx := context.Background()
	counters, err := a.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic)).
		Order(gen.Asc(counter.FieldCreatedAt)).
		Limit(a.maxResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listcounter: %w", err)
	}
	result := make([]*model.Counter, len(counters))
	for i, c := range counters {
		result[i] = &model.Counter{
			ID:        c.ID,
			UID:       c.UID,
			Topic:     c.Topic,
			Flag:      c.Flag,
			Digit:     c.Digit,
			Status:    c.Status,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) GetCounter(id int64) (model.Counter, error) {
	ctx := context.Background()
	c, err := a.client.Counter.Query().Where(counter.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Counter{}, types.ErrNotFound
		}
		return model.Counter{}, fmt.Errorf("postgres: getcounter: %w", err)
	}
	return entCounterToModel(c), nil
}

func (a *adapter) GetCounterByFlag(uid types.Uid, topic string, flag string) (model.Counter, error) {
	ctx := context.Background()
	c, err := a.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic), counter.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.Counter{}, types.ErrNotFound
		}
		return model.Counter{}, fmt.Errorf("postgres: getcounterbyflag: %w", err)
	}
	return entCounterToModel(c), nil
}

func (a *adapter) record(ctx context.Context, id, digit int64) error {
	_, err := a.client.CounterRecord.Create().
		SetCounterID(id).
		SetDigit(int32(digit)).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: counterrecord: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Bot
// ---------------------------------------------------------------------------

func (a *adapter) GetBot(id int64) (*model.Bot, error) {
	ctx := context.Background()
	b, err := a.client.Bot.Query().Where(bot.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbot: %w", err)
	}
	return entBotToModel(b), nil
}

func (a *adapter) GetBotByName(name string) (*model.Bot, error) {
	ctx := context.Background()
	b, err := a.client.Bot.Query().Where(bot.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbotbyname: %w", err)
	}
	return entBotToModel(b), nil
}

func (a *adapter) CreateBot(botModel *model.Bot) (int64, error) {
	ctx := context.Background()
	b, err := a.client.Bot.Create().
		SetName(botModel.Name).
		SetState(int(botModel.State)).
		SetCreatedAt(botModel.CreatedAt).
		SetUpdatedAt(botModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createbot: %w", err)
	}
	return b.ID, nil
}

func (a *adapter) UpdateBot(botModel *model.Bot) error {
	ctx := context.Background()
	_, err := a.client.Bot.Update().Where(bot.IDEQ(botModel.ID)).
		SetName(botModel.Name).
		SetState(int(botModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updatebot: %w", err)
	}
	return nil
}

func (a *adapter) DeleteBot(name string) error {
	ctx := context.Background()
	_, err := a.client.Bot.Delete().Where(bot.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletebot: %w", err)
	}
	return nil
}

func (a *adapter) GetBots() ([]*model.Bot, error) {
	ctx := context.Background()
	bots, err := a.client.Bot.Query().Order(gen.Asc(bot.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getbots: %w", err)
	}
	result := make([]*model.Bot, len(bots))
	for i, b := range bots {
		result[i] = entBotToModel(b)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Platform
// ---------------------------------------------------------------------------

func (a *adapter) GetPlatform(id int64) (*model.Platform, error) {
	ctx := context.Background()
	p, err := a.client.Platform.Query().Where(platform.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatform: %w", err)
	}
	return entPlatformToModel(p), nil
}

func (a *adapter) GetPlatformByName(name string) (*model.Platform, error) {
	ctx := context.Background()
	p, err := a.client.Platform.Query().Where(platform.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatformbyname: %w", err)
	}
	return entPlatformToModel(p), nil
}

func (a *adapter) GetPlatforms() ([]*model.Platform, error) {
	ctx := context.Background()
	platforms, err := a.client.Platform.Query().Order(gen.Asc(platform.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getplatforms: %w", err)
	}
	result := make([]*model.Platform, len(platforms))
	for i, p := range platforms {
		result[i] = entPlatformToModel(p)
	}
	return result, nil
}

func (a *adapter) CreatePlatform(platformModel *model.Platform) (int64, error) {
	ctx := context.Background()
	p, err := a.client.Platform.Create().
		SetName(platformModel.Name).
		SetCreatedAt(platformModel.CreatedAt).
		SetUpdatedAt(platformModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createplatform: %w", err)
	}
	return p.ID, nil
}

// ---------------------------------------------------------------------------
// Channel
// ---------------------------------------------------------------------------

func (a *adapter) GetChannel(id int64) (*model.Channel, error) {
	ctx := context.Background()
	c, err := a.client.Channel.Query().Where(channel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannel: %w", err)
	}
	return entChannelToModel(c), nil
}

func (a *adapter) GetChannelByName(name string) (*model.Channel, error) {
	ctx := context.Background()
	c, err := a.client.Channel.Query().Where(channel.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannelbyname: %w", err)
	}
	return entChannelToModel(c), nil
}

func (a *adapter) CreateChannel(channelModel *model.Channel) (int64, error) {
	ctx := context.Background()
	c, err := a.client.Channel.Create().
		SetName(channelModel.Name).
		SetFlag(channelModel.Flag).
		SetState(int(channelModel.State)).
		SetCreatedAt(channelModel.CreatedAt).
		SetUpdatedAt(channelModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createchannel: %w", err)
	}
	return c.ID, nil
}

func (a *adapter) UpdateChannel(channelModel *model.Channel) error {
	ctx := context.Background()
	_, err := a.client.Channel.Update().Where(channel.IDEQ(channelModel.ID)).
		SetName(channelModel.Name).
		SetFlag(channelModel.Flag).
		SetState(int(channelModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updatechannel: %w", err)
	}
	return nil
}

func (a *adapter) DeleteChannel(name string) error {
	ctx := context.Background()
	_, err := a.client.Channel.Delete().Where(channel.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletechannel: %w", err)
	}
	return nil
}

func (a *adapter) GetChannels() ([]*model.Channel, error) {
	ctx := context.Background()
	channels, err := a.client.Channel.Query().Order(gen.Asc(channel.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getchannels: %w", err)
	}
	result := make([]*model.Channel, len(channels))
	for i, c := range channels {
		result[i] = entChannelToModel(c)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

func (a *adapter) GetAgents() ([]*model.Agent, error) {
	ctx := context.Background()
	agents, err := a.client.Agent.Query().Order(gen.Asc(agent.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getagents: %w", err)
	}
	result := make([]*model.Agent, len(agents))
	for i, ag := range agents {
		result[i] = entAgentToModel(ag)
	}
	return result, nil
}

func (a *adapter) GetAgentByHostid(uid types.Uid, topic string, hostid string) (*model.Agent, error) {
	ctx := context.Background()
	ag, err := a.client.Agent.Query().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getagentbyhostid: %w", err)
	}
	return entAgentToModel(ag), nil
}

func (a *adapter) CreateAgent(agentModel *model.Agent) (int64, error) {
	ctx := context.Background()
	ag, err := a.client.Agent.Create().
		SetUID(agentModel.UID).
		SetTopic(agentModel.Topic).
		SetHostid(agentModel.Hostid).
		SetHostname(agentModel.Hostname).
		SetOnlineDuration(agentModel.OnlineDuration).
		SetLastOnlineAt(agentModel.LastOnlineAt).
		SetCreatedAt(agentModel.CreatedAt).
		SetUpdatedAt(agentModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createagent: %w", err)
	}
	return ag.ID, nil
}

func (a *adapter) UpdateAgentLastOnlineAt(uid types.Uid, topic string, hostid string, lastOnlineAt time.Time) error {
	ctx := context.Background()
	_, err := a.client.Agent.Update().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		SetLastOnlineAt(lastOnlineAt).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updateagentlastonlineat: %w", err)
	}
	return nil
}

func (a *adapter) UpdateAgentOnlineDuration(uid types.Uid, topic string, hostid string, offlineTime time.Time) error {
	ctx := context.Background()
	ag, err := a.client.Agent.Query().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updateagentonlineduration query: %w", err)
	}

	onlineDuration := int32(offlineTime.Sub(ag.LastOnlineAt).Seconds())
	_, err = a.client.Agent.Update().
		Where(agent.IDEQ(ag.ID)).
		AddOnlineDuration(onlineDuration).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: updateagentonlineduration: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func entUserToModel(u *gen.User) *model.User {
	return &model.User{
		ID:        u.ID,
		Flag:      u.Flag,
		Name:      u.Name,
		Tags:      u.Tags,
		State:     model.UserState(u.State),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func entFileuploadToFileDef(f *gen.Fileupload) *types.FileDef {
	return &types.FileDef{
		ObjHeader: types.ObjHeader{
			Id:        f.Fid,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		},
		Name:     f.Name,
		MimeType: f.Mimetype,
		Size:     f.Size,
		Location: f.Location,
		User:     f.UID,
	}
}

func entPlatformUserToModel(u *gen.PlatformUser) *model.PlatformUser {
	return &model.PlatformUser{
		ID:         u.ID,
		PlatformID: u.PlatformID,
		UserID:     u.UserID,
		Flag:       u.Flag,
		Name:       u.Name,
		Email:      u.Email,
		AvatarURL:  u.AvatarURL,
		IsBot:      u.IsBot,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

func entPlatformChannelToModel(c *gen.PlatformChannel) *model.PlatformChannel {
	return &model.PlatformChannel{
		ID:         c.ID,
		PlatformID: c.PlatformID,
		ChannelID:  c.ChannelID,
		Flag:       c.Flag,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func entPlatformChannelUserToModel(u *gen.PlatformChannelUser) *model.PlatformChannelUser {
	return &model.PlatformChannelUser{
		ID:          u.ID,
		PlatformID:  u.PlatformID,
		ChannelFlag: u.ChannelFlag,
		UserFlag:    u.UserFlag,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func entMessageToModel(m *gen.Message) *model.Message {
	result := &model.Message{
		ID:            m.ID,
		Flag:          m.Flag,
		PlatformID:    m.PlatformID,
		PlatformMsgID: m.PlatformMsgID,
		Topic:         m.Topic,
		Role:          m.Role,
		Session:       m.Session,
		Content:       model.JSON(m.Content),
		State:         model.MessageState(m.State),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
	return result
}

func entOAuthToModel(o *gen.OAuth) model.OAuth {
	return model.OAuth{
		ID:        o.ID,
		UID:       o.UID,
		Topic:     o.Topic,
		Name:      o.Name,
		Type:      o.Type,
		Token:     o.Token,
		Extra:     model.JSON(o.Extra),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

func entFormToModel(f *gen.Form) model.Form {
	return model.Form{
		ID:        f.ID,
		FormID:    f.FormID,
		UID:       f.UID,
		Topic:     f.Topic,
		Schema:    model.JSON(f.Schema),
		Values:    model.JSON(f.Values),
		Extra:     model.JSON(f.Extra),
		State:     model.FormState(f.State),
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

func entPageToModel(p *gen.Page) model.Page {
	return model.Page{
		ID:        p.ID,
		PageID:    p.PageID,
		UID:       p.UID,
		Topic:     p.Topic,
		Type:      model.PageType(p.Type),
		Schema:    model.JSON(p.Schema),
		State:     model.PageState(p.State),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func entBehaviorToModel(b *gen.Behavior) model.Behavior {
	result := model.Behavior{
		ID:        b.ID,
		UID:       b.UID,
		Flag:      b.Flag,
		Count_:    b.Count,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
	if b.Extra != nil {
		extra := model.JSON(b.Extra)
		result.Extra = &extra
	}
	return result
}

func entWebhookToModel(w *gen.Webhook) *model.Webhook {
	return &model.Webhook{
		ID:           w.ID,
		UID:          w.UID,
		Topic:        w.Topic,
		Flag:         w.Flag,
		Secret:       w.Secret,
		TriggerCount: w.TriggerCount,
		State:        model.WebhookState(w.State),
		CreatedAt:    w.CreatedAt,
		UpdatedAt:    w.UpdatedAt,
	}
}

func entCounterToModel(c *gen.Counter) model.Counter {
	return model.Counter{
		ID:        c.ID,
		UID:       c.UID,
		Topic:     c.Topic,
		Flag:      c.Flag,
		Digit:     c.Digit,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func entBotToModel(b *gen.Bot) *model.Bot {
	return &model.Bot{
		ID:        b.ID,
		Name:      b.Name,
		State:     model.BotState(b.State),
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func entPlatformToModel(p *gen.Platform) *model.Platform {
	return &model.Platform{
		ID:        p.ID,
		Name:      p.Name,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func entChannelToModel(c *gen.Channel) *model.Channel {
	return &model.Channel{
		ID:        c.ID,
		Name:      c.Name,
		Flag:      c.Flag,
		State:     model.ChannelState(c.State),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func entAgentToModel(a *gen.Agent) *model.Agent {
	return &model.Agent{
		ID:             a.ID,
		UID:            a.UID,
		Topic:          a.Topic,
		Hostid:         a.Hostid,
		Hostname:       a.Hostname,
		OnlineDuration: a.OnlineDuration,
		LastOnlineAt:   a.LastOnlineAt,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

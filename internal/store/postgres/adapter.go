// Package postgres implements the PostgreSQL storage adapter.
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
	"github.com/flowline-io/flowbot/internal/store/ent/gen/configdata"
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
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
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

func (*adapter) GetName() string {
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

func (a *adapter) UserCreate(ctx context.Context, usr *gen.User) error {
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

func (a *adapter) UserGet(ctx context.Context, uid types.Uid) (*gen.User, error) {
	u, err := a.client.User.Query().Where(user.FlagEQ(uid.String())).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user: %w", err)
	}
	return u, nil
}

func (a *adapter) UserGetAll(ctx context.Context, ids ...types.Uid) ([]*gen.User, error) {
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
	result := make([]*gen.User, len(users))
	for i, u := range users {
		result[i] = u
	}
	return result, nil
}

func (a *adapter) FirstUser(ctx context.Context) (*gen.User, error) {
	u, err := a.client.User.Query().Order(gen.Asc(user.FieldID)).First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: first user: %w", err)
	}
	return u, nil
}

func (a *adapter) UserDelete(ctx context.Context, uid types.Uid, hard bool) error {
	if hard {
		_, err := a.client.User.Delete().Where(user.FlagEQ(uid.String())).Exec(ctx)
		if err != nil {
			return fmt.Errorf("postgres: hard delete user: %w", err)
		}
		return nil
	}
	_, err := a.client.User.Update().Where(user.FlagEQ(uid.String())).
		SetState(int(int(schema.UserInactive))).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: soft delete user: %w", err)
	}
	return nil
}

func (a *adapter) UserUpdate(ctx context.Context, uid types.Uid, update types.KV) error {
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

func (a *adapter) FileStartUpload(ctx context.Context, fd *types.FileDef) error {
	_, err := a.client.Fileupload.Create().
		SetUID(fd.User).
		SetFid(fd.Id).
		SetName(fd.Name).
		SetMimetype(fd.MimeType).
		SetSize(fd.Size).
		SetLocation(fd.Location).
		SetState(int(schema.FileStart)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: file start upload: %w", err)
	}
	return nil
}

func (a *adapter) FileFinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	st := int(schema.FileFailed)
	if success {
		st = int(schema.FileFinish)
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
	return a.FileGet(ctx, fd.Id)
}

func (a *adapter) FileGet(ctx context.Context, fid string) (*types.FileDef, error) {
	u, err := a.client.Fileupload.Query().Where(fileupload.FidEQ(fid)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: file get: %w", err)
	}
	return entFileuploadToFileDef(u), nil
}

func (a *adapter) FileDeleteUnused(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	q := a.client.Fileupload.Query().
		Where(fileupload.StateEQ(int(schema.FileFinish)))
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

func (a *adapter) GetUsers(ctx context.Context) ([]*gen.User, error) {
	return a.UserGetAll(ctx)
}

func (a *adapter) GetUserById(ctx context.Context, id int64) (*gen.User, error) {
	u, err := a.client.User.Query().Where(user.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user by id: %w", err)
	}
	return u, nil
}

func (a *adapter) GetUserByFlag(ctx context.Context, flag string) (*gen.User, error) {
	return a.UserGet(ctx, types.Uid(flag))
}

// ---------------------------------------------------------------------------
// PlatformUser
// ---------------------------------------------------------------------------

func (a *adapter) CreatePlatformUser(ctx context.Context, item *gen.PlatformUser) (int64, error) {
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

func (a *adapter) GetPlatformUsersByUserId(ctx context.Context, userId int64) ([]*gen.PlatformUser, error) {
	users, err := a.client.PlatformUser.Query().Where(platformuser.UserIDEQ(userId)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform users by user id: %w", err)
	}
	result := make([]*gen.PlatformUser, len(users))
	for i, u := range users {
		result[i] = u
	}
	return result, nil
}

func (a *adapter) GetPlatformUserByFlag(ctx context.Context, flag string) (*gen.PlatformUser, error) {
	u, err := a.client.PlatformUser.Query().Where(platformuser.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform user by flag: %w", err)
	}
	return u, nil
}

func (a *adapter) UpdatePlatformUser(ctx context.Context, item *gen.PlatformUser) error {
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

func (a *adapter) GetPlatformChannelByFlag(ctx context.Context, flag string) (*gen.PlatformChannel, error) {
	u, err := a.client.PlatformChannel.Query().Where(platformchannel.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by flag: %w", err)
	}
	return u, nil
}

func (a *adapter) GetPlatformChannelsByPlatformIds(ctx context.Context, platformIds []int64) ([]*gen.PlatformChannel, error) {
	channels, err := a.client.PlatformChannel.Query().Where(platformchannel.PlatformIDIn(platformIds...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channels by platform ids: %w", err)
	}
	result := make([]*gen.PlatformChannel, len(channels))
	for i, c := range channels {
		result[i] = c
	}
	return result, nil
}

func (a *adapter) GetPlatformChannelsByChannelId(ctx context.Context, channelId int64) (*gen.PlatformChannel, error) {
	u, err := a.client.PlatformChannel.Query().Where(platformchannel.ChannelIDEQ(channelId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by channel id: %w", err)
	}
	return u, nil
}

func (a *adapter) CreatePlatformChannel(ctx context.Context, item *gen.PlatformChannel) (int64, error) {
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

func (a *adapter) CreatePlatformChannelUser(ctx context.Context, item *gen.PlatformChannelUser) (int64, error) {
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

func (a *adapter) GetPlatformChannelUsersByUserFlag(ctx context.Context, userFlag string) ([]*gen.PlatformChannelUser, error) {
	users, err := a.client.PlatformChannelUser.Query().
		Where(platformchanneluser.UserFlagEQ(userFlag)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channel users by user flag: %w", err)
	}
	result := make([]*gen.PlatformChannelUser, len(users))
	for i, u := range users {
		result[i] = u
	}
	return result, nil
}

// GetPlatformChannelUsersByUserFlags returns platform channel user records for a batch of user flags.
func (a *adapter) GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error) {
	if len(userFlags) == 0 {
		return nil, nil
	}
	users, err := a.client.PlatformChannelUser.Query().
		Where(platformchanneluser.UserFlagIn(userFlags...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channel users by user flags: %w", err)
	}
	result := make([]*gen.PlatformChannelUser, len(users))
	for i, u := range users {
		result[i] = u
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

func (a *adapter) GetMessage(ctx context.Context, flag string) (*gen.Message, error) {
	m, err := a.client.Message.Query().Where(message.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message: %w", err)
	}
	return m, nil
}

func (a *adapter) GetMessageByPlatform(ctx context.Context, platformId int64, platformMsgId string) (*gen.Message, error) {
	m, err := a.client.Message.Query().
		Where(message.PlatformIDEQ(platformId), message.PlatformMsgIDEQ(platformMsgId)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message by platform: %w", err)
	}
	return m, nil
}

func (a *adapter) GetMessagesBySession(ctx context.Context, session string) ([]*gen.Message, error) {
	messages, err := a.client.Message.Query().
		Where(message.SessionEQ(session)).
		Order(gen.Asc(message.FieldCreatedAt)).
		Limit(a.maxMessageResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get messages by session: %w", err)
	}
	result := make([]*gen.Message, len(messages))
	for i, m := range messages {
		result[i] = m
	}
	return result, nil
}

func (a *adapter) CreateMessage(ctx context.Context, msg gen.Message) error {
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

func (a *adapter) DataSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error {
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

func (a *adapter) DataGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
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

func (a *adapter) DataList(ctx context.Context, uid types.Uid, topic string, filter types.DataFilter) ([]*gen.Data, error) {
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
	result := make([]*gen.Data, len(items))
	for i, d := range items {
		result[i] = &gen.Data{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     schema.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) DataDelete(ctx context.Context, uid types.Uid, topic, key string) error {
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

func (a *adapter) ConfigSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error {
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

func (a *adapter) ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
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

func (a *adapter) ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]*gen.ConfigData, error) {
	q := a.client.ConfigData.Query().Where(configdata.UID(uid.String()), configdata.Topic(topic))
	if prefix != "" {
		q = q.Where(configdata.KeyHasPrefix(prefix))
	}
	q = q.Order(gen.Asc(configdata.FieldCreatedAt)).Limit(a.maxResults)
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigbyprefix: %w", err)
	}
	result := make([]*gen.ConfigData, len(items))
	for i, d := range items {
		result[i] = &gen.ConfigData{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     schema.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) ConfigDelete(ctx context.Context, uid types.Uid, topic, key string) error {
	_, err := a.client.ConfigData.Delete().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: configdelete: %w", err)
	}
	return nil
}

func (a *adapter) ListConfigs(ctx context.Context, opts store.ListConfigOptions) ([]model.ConfigItem, error) {
	q := a.client.ConfigData.Query()
	if opts.Search != "" {
		q = q.Where(
			configdata.Or(
				configdata.UIDContains(opts.Search),
				configdata.TopicContains(opts.Search),
				configdata.KeyContains(opts.Search),
			),
		)
	}
	limit := opts.Limit
	if limit <= 0 || limit > a.maxResults {
		limit = a.maxResults
	}
	items, err := q.
		Offset(opts.Offset).
		Limit(limit).
		Order(gen.Desc(configdata.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigs: %w", err)
	}
	result := make([]model.ConfigItem, len(items))
	for i, d := range items {
		result[i] = model.ConfigItem{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     types.KV(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// OAuth
// ---------------------------------------------------------------------------

func (a *adapter) OAuthSet(ctx context.Context, oauthModel gen.OAuth) error {
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
		if oauthModel.RefreshToken != "" {
			u = u.SetRefreshToken(oauthModel.RefreshToken)
		}
		if !oauthModel.ExpiresAt.IsZero() {
			u = u.SetExpiresAt(oauthModel.ExpiresAt)
		}
		if oauthModel.TokenType != "" {
			u = u.SetTokenType(oauthModel.TokenType)
		}
		if oauthModel.Scope != "" {
			u = u.SetScope(oauthModel.Scope)
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
		if oauthModel.RefreshToken != "" {
			c = c.SetRefreshToken(oauthModel.RefreshToken)
		}
		if !oauthModel.ExpiresAt.IsZero() {
			c = c.SetExpiresAt(oauthModel.ExpiresAt)
		}
		if oauthModel.TokenType != "" {
			c = c.SetTokenType(oauthModel.TokenType)
		}
		if oauthModel.Scope != "" {
			c = c.SetScope(oauthModel.Scope)
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: oauthset save: %w", err)
	}
	return nil
}

func (a *adapter) OAuthGet(ctx context.Context, uid types.Uid, topic, t string) (gen.OAuth, error) {
	o, err := a.client.OAuth.Query().
		Where(oauth.UID(uid.String()), oauth.Topic(topic), oauth.Type(t)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.OAuth{}, types.ErrNotFound
		}
		return gen.OAuth{}, fmt.Errorf("postgres: oauthget: %w", err)
	}
	return *o, nil
}

func (a *adapter) OAuthGetAvailable(ctx context.Context, t string) ([]gen.OAuth, error) {
	oauths, err := a.client.OAuth.Query().Where(oauth.Type(t)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: oauthgetavailable: %w", err)
	}
	result := make([]gen.OAuth, len(oauths))
	for i, o := range oauths {
		result[i] = *o
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

func (a *adapter) FormSet(ctx context.Context, formId string, formModel gen.Form) error {
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

func (a *adapter) FormGet(ctx context.Context, formId string) (gen.Form, error) {
	f, err := a.client.Form.Query().Where(form.FormIDEQ(formId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Form{}, types.ErrNotFound
		}
		return gen.Form{}, fmt.Errorf("postgres: formget: %w", err)
	}
	return *f, nil
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

func (a *adapter) PageSet(ctx context.Context, pageId string, pageModel gen.Page) error {
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

func (a *adapter) PageGet(ctx context.Context, pageId string) (gen.Page, error) {
	p, err := a.client.Page.Query().Where(page.PageIDEQ(pageId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Page{}, types.ErrNotFound
		}
		return gen.Page{}, fmt.Errorf("postgres: pageget: %w", err)
	}
	return *p, nil
}

// ---------------------------------------------------------------------------
// Behavior
// ---------------------------------------------------------------------------

func (a *adapter) BehaviorSet(ctx context.Context, behaviorModel gen.Behavior) error {
	existing, err := a.client.Behavior.Query().
		Where(behavior.UID(behaviorModel.UID), behavior.Flag(behaviorModel.Flag)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: behaviorset query: %w", err)
	}

	if existing != nil {
		u := a.client.Behavior.Update().Where(behavior.IDEQ(existing.ID)).
			SetCount(behaviorModel.Count).
			SetUpdatedAt(time.Now())
		if behaviorModel.Extra != nil {
			u = u.SetExtra(behaviorModel.Extra)
		}
		_, err = u.Save(ctx)
	} else {
		c := a.client.Behavior.Create().
			SetUID(behaviorModel.UID).
			SetFlag(behaviorModel.Flag).
			SetCount(behaviorModel.Count).
			SetCreatedAt(behaviorModel.CreatedAt).
			SetUpdatedAt(behaviorModel.UpdatedAt)
		if behaviorModel.Extra != nil {
			c = c.SetExtra(behaviorModel.Extra)
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: behaviorset save: %w", err)
	}
	return nil
}

func (a *adapter) BehaviorGet(ctx context.Context, uid types.Uid, flag string) (gen.Behavior, error) {
	b, err := a.client.Behavior.Query().
		Where(behavior.UID(uid.String()), behavior.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Behavior{}, types.ErrNotFound
		}
		return gen.Behavior{}, fmt.Errorf("postgres: behaviorget: %w", err)
	}
	return *b, nil
}

func (a *adapter) BehaviorList(ctx context.Context, uid types.Uid) ([]*gen.Behavior, error) {
	behaviors, err := a.client.Behavior.Query().
		Where(behavior.UID(uid.String())).
		Order(gen.Asc(behavior.FieldCreatedAt)).
		Limit(a.maxResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: behaviorlist: %w", err)
	}
	return behaviors, nil
}

func (a *adapter) BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error {
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

func (a *adapter) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
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

func (a *adapter) ParameterGet(ctx context.Context, flag string) (gen.Parameter, error) {
	p, err := a.client.Parameter.Query().Where(parameter.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Parameter{}, types.ErrNotFound
		}
		return gen.Parameter{}, fmt.Errorf("postgres: parameterget: %w", err)
	}
	return gen.Parameter{
		ID:        p.ID,
		Flag:      p.Flag,
		Params:    schema.JSON(p.Params),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		ExpiredAt: p.ExpiredAt,
	}, nil
}

func (a *adapter) ParameterDelete(ctx context.Context, flag string) error {
	_, err := a.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: parameterdelete: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Instruct
// ---------------------------------------------------------------------------

func (a *adapter) CreateInstruct(ctx context.Context, instructModel *gen.Instruct) (int64, error) {
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

func (a *adapter) ListInstruct(ctx context.Context, uid types.Uid, isExpire bool, limit int) ([]*gen.Instruct, error) {
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
	return items, nil
}

func (a *adapter) UpdateInstruct(ctx context.Context, instructModel *gen.Instruct) error {
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
// Counter
// ---------------------------------------------------------------------------

func (a *adapter) CreateCounter(ctx context.Context, counterModel *gen.Counter) (int64, error) {
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

func (a *adapter) IncreaseCounter(ctx context.Context, id, amount int64) error {
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

func (a *adapter) DecreaseCounter(ctx context.Context, id, amount int64) error {
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

func (a *adapter) ListCounter(ctx context.Context, uid types.Uid, topic string) ([]*gen.Counter, error) {
	counters, err := a.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic)).
		Order(gen.Asc(counter.FieldCreatedAt)).
		Limit(a.maxResults).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listcounter: %w", err)
	}
	return counters, nil
}

func (a *adapter) GetCounter(ctx context.Context, id int64) (gen.Counter, error) {
	c, err := a.client.Counter.Query().Where(counter.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Counter{}, types.ErrNotFound
		}
		return gen.Counter{}, fmt.Errorf("postgres: getcounter: %w", err)
	}
	return *c, nil
}

func (a *adapter) GetCounterByFlag(ctx context.Context, uid types.Uid, topic, flag string) (gen.Counter, error) {
	c, err := a.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic), counter.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Counter{}, types.ErrNotFound
		}
		return gen.Counter{}, fmt.Errorf("postgres: getcounterbyflag: %w", err)
	}
	return *c, nil
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

func (a *adapter) GetBot(ctx context.Context, id int64) (*gen.Bot, error) {
	b, err := a.client.Bot.Query().Where(bot.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbot: %w", err)
	}
	return b, nil
}

func (a *adapter) GetBotByName(ctx context.Context, name string) (*gen.Bot, error) {
	b, err := a.client.Bot.Query().Where(bot.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbotbyname: %w", err)
	}
	return b, nil
}

func (a *adapter) CreateBot(ctx context.Context, botModel *gen.Bot) (int64, error) {
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

func (a *adapter) UpdateBot(ctx context.Context, botModel *gen.Bot) error {
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

func (a *adapter) DeleteBot(ctx context.Context, name string) error {
	_, err := a.client.Bot.Delete().Where(bot.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletebot: %w", err)
	}
	return nil
}

func (a *adapter) GetBots(ctx context.Context) ([]*gen.Bot, error) {
	bots, err := a.client.Bot.Query().Order(gen.Asc(bot.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getbots: %w", err)
	}
	result := make([]*gen.Bot, len(bots))
	for i, b := range bots {
		result[i] = b
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Platform
// ---------------------------------------------------------------------------

func (a *adapter) GetPlatform(ctx context.Context, id int64) (*gen.Platform, error) {
	p, err := a.client.Platform.Query().Where(platform.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatform: %w", err)
	}
	return p, nil
}

func (a *adapter) GetPlatformByName(ctx context.Context, name string) (*gen.Platform, error) {
	p, err := a.client.Platform.Query().Where(platform.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatformbyname: %w", err)
	}
	return p, nil
}

func (a *adapter) GetPlatforms(ctx context.Context) ([]*gen.Platform, error) {
	platforms, err := a.client.Platform.Query().Order(gen.Asc(platform.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getplatforms: %w", err)
	}
	result := make([]*gen.Platform, len(platforms))
	for i, p := range platforms {
		result[i] = p
	}
	return result, nil
}

func (a *adapter) CreatePlatform(ctx context.Context, platformModel *gen.Platform) (int64, error) {
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

func (a *adapter) GetChannel(ctx context.Context, id int64) (*gen.Channel, error) {
	c, err := a.client.Channel.Query().Where(channel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannel: %w", err)
	}
	return c, nil
}

func (a *adapter) GetChannelByName(ctx context.Context, name string) (*gen.Channel, error) {
	c, err := a.client.Channel.Query().Where(channel.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannelbyname: %w", err)
	}
	return c, nil
}

func (a *adapter) CreateChannel(ctx context.Context, channelModel *gen.Channel) (int64, error) {
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

func (a *adapter) UpdateChannel(ctx context.Context, channelModel *gen.Channel) error {
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

func (a *adapter) DeleteChannel(ctx context.Context, name string) error {
	_, err := a.client.Channel.Delete().Where(channel.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletechannel: %w", err)
	}
	return nil
}

func (a *adapter) GetChannels(ctx context.Context) ([]*gen.Channel, error) {
	channels, err := a.client.Channel.Query().Order(gen.Asc(channel.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getchannels: %w", err)
	}
	result := make([]*gen.Channel, len(channels))
	for i, c := range channels {
		result[i] = c
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

func (a *adapter) GetAgents(ctx context.Context) ([]*gen.Agent, error) {
	agents, err := a.client.Agent.Query().Order(gen.Asc(agent.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getagents: %w", err)
	}
	result := make([]*gen.Agent, len(agents))
	for i, ag := range agents {
		result[i] = ag
	}
	return result, nil
}

func (a *adapter) GetAgentByHostid(ctx context.Context, uid types.Uid, topic, hostid string) (*gen.Agent, error) {
	ag, err := a.client.Agent.Query().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getagentbyhostid: %w", err)
	}
	return ag, nil
}

func (a *adapter) CreateAgent(ctx context.Context, agentModel *gen.Agent) (int64, error) {
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

func (a *adapter) UpdateAgentLastOnlineAt(ctx context.Context, uid types.Uid, topic, hostid string, lastOnlineAt time.Time) error {
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

func (a *adapter) UpdateAgentOnlineDuration(ctx context.Context, uid types.Uid, topic, hostid string, offlineTime time.Time) error {
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

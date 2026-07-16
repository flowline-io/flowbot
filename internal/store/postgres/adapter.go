// Package postgres implements the PostgreSQL storage adapter.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	_ "github.com/jackc/pgx/v5/stdlib" //revive:disable:blank-imports pgx driver registration

	entsql "entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentplan"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentskill"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentskillfile"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsubagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsubagenttask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/behavior"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/bot"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/channel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtaskrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsession"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsessionentry"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/configdata"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/counter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/data"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/fileupload"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/form"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/instruct"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/message"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifychannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifyrule"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/oauth"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/page"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platform"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchanneluser"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformuser"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/user"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
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

// Ping checks PostgreSQL connectivity and returns the round-trip latency.
func (a *adapter) Ping(ctx context.Context) (time.Duration, error) {
	if a.db == nil {
		return 0, fmt.Errorf("postgres: database not initialized")
	}
	start := time.Now()
	err := a.db.PingContext(ctx)
	return time.Since(start), err
}

func (a *adapter) GetDB() any {
	return a.client
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

func (a *adapter) UserCreate(ctx context.Context, usr *gen.User) error {
	u, err := a.client.User.Create().
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
	usr.ID = u.ID
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
	copy(result, users)
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

// normalizePlatformUserProfile fills required profile fields when callers omit them.
func normalizePlatformUserProfile(item *gen.PlatformUser) {
	if item == nil {
		return
	}
	if item.Email == "" {
		flag := item.Flag
		if flag == "" {
			flag = "user"
		}
		item.Email = fmt.Sprintf("%s@unknown.local", flag)
	}
	if item.AvatarURL == "" {
		item.AvatarURL = "-"
	}
}

func (a *adapter) CreatePlatformUser(ctx context.Context, item *gen.PlatformUser) (int64, error) {
	normalizePlatformUserProfile(item)
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
	copy(result, users)
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
	copy(result, channels)
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

func (a *adapter) UpdatePlatformChannelChannelID(ctx context.Context, platformChannelID, channelID int64) error {
	n, err := a.client.PlatformChannel.Update().
		Where(platformchannel.IDEQ(platformChannelID)).
		SetChannelID(channelID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update platform channel channel id: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
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
	copy(result, users)
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
	copy(result, users)
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
	copy(result, messages)
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

func (a *adapter) CreateChatSession(ctx context.Context, session *gen.ChatSession) error {
	if session == nil {
		return errors.New("postgres: nil chat session")
	}
	builder := a.client.ChatSession.Create().
		SetFlag(session.Flag).
		SetUID(session.UID).
		SetLeafID(session.LeafID).
		SetState(session.State)
	if !session.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(session.CreatedAt)
	}
	if !session.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(session.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat session: %w", err)
	}
	return nil
}

func (a *adapter) GetChatSession(ctx context.Context, flag string) (*gen.ChatSession, error) {
	row, err := a.client.ChatSession.Query().
		Where(chatsession.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session: %w", err)
	}
	return row, nil
}

func (a *adapter) ListChatSessions(ctx context.Context, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := a.client.ChatSession.Query().
		Order(
			gen.Desc(chatsession.FieldUpdatedAt),
			gen.Desc(chatsession.FieldID),
		).
		Limit(opts.Limit + 1)

	if opts.Cursor != "" {
		id, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil {
			q = q.Where(chatsession.IDLT(id))
		}
	}
	if opts.UID != "" {
		q = q.Where(chatsession.UIDEQ(opts.UID))
	}
	if opts.State != nil {
		q = q.Where(chatsession.StateEQ(*opts.State))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: list chat sessions: %w", err)
	}

	var nextCursor string
	if len(rows) > opts.Limit {
		nextCursor = strconv.FormatInt(rows[opts.Limit-1].ID, 10)
		rows = rows[:opts.Limit]
	}

	return rows, nextCursor, nil
}

func (a *adapter) UpdateChatSessionLeaf(ctx context.Context, flag, leafID string) error {
	n, err := a.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetLeafID(leafID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session leaf: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) UpdateChatSessionMode(ctx context.Context, flag, mode string) error {
	n, err := a.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetMode(mode).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session mode: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) UpdateChatSessionTitle(ctx context.Context, flag, title string) error {
	n, err := a.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetTitle(title).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session title: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) CloseChatSession(ctx context.Context, flag string) error {
	_, err := a.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetState(int(schema.ChatSessionClosed)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: close chat session: %w", err)
	}
	return nil
}

func (a *adapter) CreateChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error {
	if entry == nil {
		return errors.New("postgres: nil chat session entry")
	}
	builder := a.client.ChatSessionEntry.Create().
		SetFlag(entry.Flag).
		SetSessionID(entry.SessionID).
		SetParentID(entry.ParentID).
		SetEntryType(entry.EntryType)
	if entry.Payload != nil {
		builder = builder.SetPayload(entry.Payload)
	}
	if !entry.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(entry.CreatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat session entry: %w", err)
	}
	return nil
}

func (a *adapter) AppendChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error {
	if entry == nil {
		return errors.New("postgres: nil chat session entry")
	}
	tx, err := a.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin chat session tx: %w", err)
	}
	builder := tx.ChatSessionEntry.Create().
		SetFlag(entry.Flag).
		SetSessionID(entry.SessionID).
		SetParentID(entry.ParentID).
		SetEntryType(entry.EntryType)
	if entry.Payload != nil {
		builder = builder.SetPayload(entry.Payload)
	}
	if !entry.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(entry.CreatedAt)
	}
	if _, err := builder.Save(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: create chat session entry: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: create chat session entry: %w", err)
	}
	n, err := tx.ChatSession.Update().
		Where(chatsession.FlagEQ(entry.SessionID)).
		SetLeafID(entry.Flag).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: update chat session leaf: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: update chat session leaf: %w", err)
	}
	if n == 0 {
		if rerr := tx.Rollback(); rerr != nil {
			return types.ErrNotFound
		}
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit chat session entry: %w", err)
	}
	return nil
}

func (a *adapter) ListChatSessionEntries(ctx context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	rows, err := a.client.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDEQ(sessionID)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat session entries: %w", err)
	}
	return rows, nil
}

func (a *adapter) GetChatSessionEntry(ctx context.Context, flag string) (*gen.ChatSessionEntry, error) {
	row, err := a.client.ChatSessionEntry.Query().
		Where(chatsessionentry.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session entry: %w", err)
	}
	return row, nil
}

func (a *adapter) GetChatSessionEntryInSession(ctx context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error) {
	row, err := a.client.ChatSessionEntry.Query().
		Where(
			chatsessionentry.SessionIDEQ(sessionID),
			chatsessionentry.FlagEQ(flag),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session entry in session: %w", err)
	}
	return row, nil
}

func (a *adapter) CreateChatScheduledTask(ctx context.Context, task *gen.ChatScheduledTask) error {
	if task == nil {
		return errors.New("postgres: nil chat scheduled task")
	}
	builder := a.client.ChatScheduledTask.Create().
		SetFlag(task.Flag).
		SetUID(task.UID).
		SetName(task.Name).
		SetScheduleKind(task.ScheduleKind).
		SetCron(task.Cron).
		SetPrompt(task.Prompt).
		SetSourceSessionID(task.SourceSessionID).
		SetState(task.State)
	if task.RunAt != nil {
		builder = builder.SetRunAt(*task.RunAt)
	}
	if task.Delivery != nil {
		builder = builder.SetDelivery(task.Delivery)
	}
	if task.LastRunAt != nil {
		builder = builder.SetLastRunAt(*task.LastRunAt)
	}
	if task.NextRunAt != nil {
		builder = builder.SetNextRunAt(*task.NextRunAt)
	}
	if !task.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(task.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat scheduled task: %w", err)
	}
	return nil
}

func (a *adapter) DeleteChatScheduledTask(ctx context.Context, flag string) error {
	n, err := a.client.ChatScheduledTask.Delete().
		Where(chatscheduledtask.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete chat scheduled task: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) GetChatScheduledTask(ctx context.Context, flag string) (*gen.ChatScheduledTask, error) {
	row, err := a.client.ChatScheduledTask.Query().
		Where(chatscheduledtask.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat scheduled task: %w", err)
	}
	return row, nil
}

func (a *adapter) GetChatScheduledTaskForUID(ctx context.Context, flag, uid string) (*gen.ChatScheduledTask, error) {
	row, err := a.client.ChatScheduledTask.Query().
		Where(
			chatscheduledtask.FlagEQ(flag),
			chatscheduledtask.UIDEQ(uid),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat scheduled task for uid: %w", err)
	}
	return row, nil
}

func (a *adapter) ListChatScheduledTasks(ctx context.Context, opts store.ListChatScheduledTasksOptions) ([]*gen.ChatScheduledTask, error) {
	q := a.client.ChatScheduledTask.Query().
		Order(
			gen.Desc(chatscheduledtask.FieldUpdatedAt),
			gen.Desc(chatscheduledtask.FieldID),
		)
	if opts.UID != "" {
		q = q.Where(chatscheduledtask.UIDEQ(opts.UID))
	}
	if len(opts.States) > 0 {
		q = q.Where(chatscheduledtask.StateIn(opts.States...))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat scheduled tasks: %w", err)
	}
	return rows, nil
}

func (a *adapter) UpdateChatScheduledTask(ctx context.Context, flag string, params store.UpdateChatScheduledTaskParams) error {
	builder := a.client.ChatScheduledTask.Update().
		Where(chatscheduledtask.FlagEQ(flag)).
		SetUpdatedAt(time.Now())
	if params.Name != nil {
		builder = builder.SetName(*params.Name)
	}
	if params.Cron != nil {
		builder = builder.SetCron(*params.Cron)
	}
	if params.RunAt != nil {
		builder = builder.SetRunAt(*params.RunAt)
	}
	if params.Prompt != nil {
		builder = builder.SetPrompt(*params.Prompt)
	}
	if params.State != nil {
		builder = builder.SetState(*params.State)
	}
	if params.LastRunAt != nil {
		builder = builder.SetLastRunAt(*params.LastRunAt)
	}
	if params.NextRunAt != nil {
		builder = builder.SetNextRunAt(*params.NextRunAt)
	}
	n, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat scheduled task: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) CreateChatScheduledTaskRun(ctx context.Context, run *gen.ChatScheduledTaskRun) error {
	if run == nil {
		return errors.New("postgres: nil chat scheduled task run")
	}
	builder := a.client.ChatScheduledTaskRun.Create().
		SetFlag(run.Flag).
		SetTaskID(run.TaskID).
		SetRunSessionID(run.RunSessionID).
		SetState(run.State).
		SetReply(run.Reply).
		SetError(run.Error)
	if !run.StartedAt.IsZero() {
		builder = builder.SetStartedAt(run.StartedAt)
	}
	if run.FinishedAt != nil {
		builder = builder.SetFinishedAt(*run.FinishedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat scheduled task run: %w", err)
	}
	return nil
}

func (a *adapter) UpdateChatScheduledTaskRun(ctx context.Context, flag string, params store.UpdateChatScheduledTaskRunParams) error {
	builder := a.client.ChatScheduledTaskRun.Update().
		Where(chatscheduledtaskrun.FlagEQ(flag))
	if params.State != nil {
		builder = builder.SetState(*params.State)
	}
	if params.Reply != nil {
		builder = builder.SetReply(*params.Reply)
	}
	if params.Error != nil {
		builder = builder.SetError(*params.Error)
	}
	if params.FinishedAt != nil {
		builder = builder.SetFinishedAt(*params.FinishedAt)
	}
	n, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat scheduled task run: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) FailStaleChatScheduledTaskRuns(ctx context.Context) error {
	now := time.Now().UTC()
	msg := "interrupted by server restart"
	_, err := a.client.ChatScheduledTaskRun.Update().
		Where(chatscheduledtaskrun.StateEQ(string(schema.ChatScheduledTaskRunStateRunning))).
		SetState(string(schema.ChatScheduledTaskRunStateFailed)).
		SetError(msg).
		SetFinishedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: fail stale chat scheduled task runs: %w", err)
	}
	return nil
}

func (a *adapter) ListChatScheduledTaskRuns(ctx context.Context, taskID string, limit int) ([]*gen.ChatScheduledTaskRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := a.client.ChatScheduledTaskRun.Query().
		Where(chatscheduledtaskrun.TaskIDEQ(taskID)).
		Order(
			gen.Desc(chatscheduledtaskrun.FieldStartedAt),
			gen.Desc(chatscheduledtaskrun.FieldID),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat scheduled task runs: %w", err)
	}
	return rows, nil
}

func (a *adapter) CreateAgentPlan(ctx context.Context, plan *gen.AgentPlan) error {
	if plan == nil {
		return errors.New("postgres: nil agent plan")
	}
	builder := a.client.AgentPlan.Create().
		SetFlag(plan.Flag).
		SetSessionID(plan.SessionID).
		SetTitle(plan.Title).
		SetContent(plan.Content).
		SetSourceEntryID(plan.SourceEntryID)
	if !plan.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(plan.CreatedAt)
	}
	if !plan.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(plan.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent plan: %w", err)
	}
	return nil
}

func (a *adapter) GetAgentPlan(ctx context.Context, flag string) (*gen.AgentPlan, error) {
	row, err := a.client.AgentPlan.Query().
		Where(agentplan.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent plan: %w", err)
	}
	return row, nil
}

func (a *adapter) GetAgentPlanInSession(ctx context.Context, sessionID, flag string) (*gen.AgentPlan, error) {
	row, err := a.client.AgentPlan.Query().
		Where(agentplan.FlagEQ(flag), agentplan.SessionIDEQ(sessionID)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent plan in session: %w", err)
	}
	return row, nil
}

func (a *adapter) ListAgentPlansBySession(ctx context.Context, sessionID string) ([]*gen.AgentPlan, error) {
	rows, err := a.client.AgentPlan.Query().
		Where(agentplan.SessionIDEQ(sessionID)).
		Order(gen.Desc(agentplan.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent plans: %w", err)
	}
	return rows, nil
}

func (a *adapter) ListAgentSkills(ctx context.Context, enabledOnly bool) ([]*gen.AgentSkill, error) {
	query := a.client.AgentSkill.Query()
	if enabledOnly {
		query = query.Where(agentskill.EnabledEQ(true))
	}
	rows, err := query.Order(gen.Asc(agentskill.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent skills: %w", err)
	}
	return rows, nil
}

func (a *adapter) GetAgentSkillsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	var maxUpdated time.Time
	row, err := a.client.AgentSkill.Query().
		Where(agentskill.EnabledEQ(true)).
		Order(gen.Desc(agentskill.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if !gen.IsNotFound(err) {
			return time.Time{}, fmt.Errorf("postgres: agent skills max updated_at: %w", err)
		}
	} else {
		maxUpdated = row.UpdatedAt
	}
	fileRow, err := a.client.AgentSkillFile.Query().
		Order(gen.Desc(agentskillfile.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return maxUpdated, nil
		}
		return time.Time{}, fmt.Errorf("postgres: agent skill files max updated_at: %w", err)
	}
	if fileRow.UpdatedAt.After(maxUpdated) {
		maxUpdated = fileRow.UpdatedAt
	}
	return maxUpdated, nil
}

func (a *adapter) GetAgentSkillByName(ctx context.Context, name string) (*gen.AgentSkill, error) {
	row, err := a.client.AgentSkill.Query().
		Where(agentskill.NameEQ(name), agentskill.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill: %w", err)
	}
	return row, nil
}

func (a *adapter) GetAgentSkillByFlag(ctx context.Context, flag string) (*gen.AgentSkill, error) {
	row, err := a.client.AgentSkill.Query().
		Where(agentskill.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill by flag: %w", err)
	}
	return row, nil
}

func (a *adapter) CreateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error {
	if skill == nil {
		return errors.New("postgres: nil agent skill")
	}
	builder := a.client.AgentSkill.Create().
		SetFlag(skill.Flag).
		SetName(skill.Name).
		SetDescription(skill.Description).
		SetContent(skill.Content).
		SetBaseDir(skill.BaseDir).
		SetSource(skill.Source).
		SetEnabled(skill.Enabled).
		SetDisableModelInvocation(skill.DisableModelInvocation)
	if !skill.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(skill.CreatedAt)
	}
	if !skill.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(skill.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent skill: %w", err)
	}
	return nil
}

func (a *adapter) UpdateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error {
	if skill == nil {
		return errors.New("postgres: nil agent skill")
	}
	n, err := a.client.AgentSkill.Update().
		Where(agentskill.FlagEQ(skill.Flag)).
		SetName(skill.Name).
		SetDescription(skill.Description).
		SetContent(skill.Content).
		SetBaseDir(skill.BaseDir).
		SetSource(skill.Source).
		SetEnabled(skill.Enabled).
		SetDisableModelInvocation(skill.DisableModelInvocation).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent skill: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteAgentSkill(ctx context.Context, flag string) error {
	tx, err := a.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin delete agent skill tx: %w", err)
	}
	if _, err := tx.AgentSkillFile.Delete().
		Where(agentskillfile.SkillFlagEQ(flag)).
		Exec(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: delete agent skill files: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: delete agent skill files: %w", err)
	}
	n, err := tx.AgentSkill.Delete().
		Where(agentskill.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: delete agent skill: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: delete agent skill: %w", err)
	}
	if n == 0 {
		if rerr := tx.Rollback(); rerr != nil {
			return types.ErrNotFound
		}
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit delete agent skill: %w", err)
	}
	return nil
}

func (a *adapter) ListAgentSkillFiles(ctx context.Context, skillFlag string) ([]*gen.AgentSkillFile, error) {
	rows, err := a.client.AgentSkillFile.Query().
		Where(agentskillfile.SkillFlagEQ(skillFlag)).
		Order(gen.Asc(agentskillfile.FieldPath)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent skill files: %w", err)
	}
	return rows, nil
}

func (a *adapter) GetAgentSkillFile(ctx context.Context, skillFlag, path string) (*gen.AgentSkillFile, error) {
	row, err := a.client.AgentSkillFile.Query().
		Where(
			agentskillfile.SkillFlagEQ(skillFlag),
			agentskillfile.PathEQ(path),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill file: %w", err)
	}
	return row, nil
}

func (a *adapter) CreateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error {
	if file == nil {
		return errors.New("postgres: nil agent skill file")
	}
	builder := a.client.AgentSkillFile.Create().
		SetSkillFlag(file.SkillFlag).
		SetPath(file.Path).
		SetContent(file.Content)
	if !file.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(file.CreatedAt)
	}
	if !file.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(file.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent skill file: %w", err)
	}
	return nil
}

func (a *adapter) UpdateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error {
	if file == nil {
		return errors.New("postgres: nil agent skill file")
	}
	n, err := a.client.AgentSkillFile.Update().
		Where(
			agentskillfile.SkillFlagEQ(file.SkillFlag),
			agentskillfile.PathEQ(file.Path),
		).
		SetContent(file.Content).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent skill file: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteAgentSkillFile(ctx context.Context, skillFlag, path string) error {
	n, err := a.client.AgentSkillFile.Delete().
		Where(
			agentskillfile.SkillFlagEQ(skillFlag),
			agentskillfile.PathEQ(path),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent skill file: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteAgentSkillFilesByFlag(ctx context.Context, skillFlag string) error {
	_, err := a.client.AgentSkillFile.Delete().
		Where(agentskillfile.SkillFlagEQ(skillFlag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent skill files by flag: %w", err)
	}
	return nil
}

func (a *adapter) ListAgentSubagents(ctx context.Context, enabledOnly bool) ([]*gen.AgentSubagent, error) {
	query := a.client.AgentSubagent.Query()
	if enabledOnly {
		query = query.Where(agentsubagent.EnabledEQ(true))
	}
	rows, err := query.Order(gen.Asc(agentsubagent.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent subagents: %w", err)
	}
	return rows, nil
}

func (a *adapter) GetAgentSubagentsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	row, err := a.client.AgentSubagent.Query().
		Where(agentsubagent.EnabledEQ(true)).
		Order(gen.Desc(agentsubagent.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("postgres: agent subagents max updated_at: %w", err)
	}
	return row.UpdatedAt, nil
}

func (a *adapter) GetAgentSubagentByName(ctx context.Context, name string) (*gen.AgentSubagent, error) {
	row, err := a.client.AgentSubagent.Query().
		Where(agentsubagent.NameEQ(name), agentsubagent.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent: %w", err)
	}
	return row, nil
}

func (a *adapter) GetAgentSubagentByFlag(ctx context.Context, flag string) (*gen.AgentSubagent, error) {
	row, err := a.client.AgentSubagent.Query().
		Where(agentsubagent.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent by flag: %w", err)
	}
	return row, nil
}

func (a *adapter) CreateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error {
	if subagent == nil {
		return errors.New("postgres: nil agent subagent")
	}
	builder := a.client.AgentSubagent.Create().
		SetFlag(subagent.Flag).
		SetName(subagent.Name).
		SetDescription(subagent.Description).
		SetSystemPrompt(subagent.SystemPrompt).
		SetTools(subagent.Tools).
		SetSkills(subagent.Skills).
		SetModel(subagent.Model).
		SetSource(subagent.Source).
		SetEnabled(subagent.Enabled)
	if !subagent.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(subagent.CreatedAt)
	}
	if !subagent.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(subagent.UpdatedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent subagent: %w", err)
	}
	subagent.ID = row.ID
	return nil
}

func (a *adapter) UpdateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error {
	if subagent == nil {
		return errors.New("postgres: nil agent subagent")
	}
	n, err := a.client.AgentSubagent.Update().
		Where(agentsubagent.FlagEQ(subagent.Flag)).
		SetName(subagent.Name).
		SetDescription(subagent.Description).
		SetSystemPrompt(subagent.SystemPrompt).
		SetTools(subagent.Tools).
		SetSkills(subagent.Skills).
		SetModel(subagent.Model).
		SetSource(subagent.Source).
		SetEnabled(subagent.Enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent subagent: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteAgentSubagent(ctx context.Context, flag string) error {
	n, err := a.client.AgentSubagent.Delete().
		Where(agentsubagent.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent subagent: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) CreateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error {
	if task == nil {
		return errors.New("postgres: nil agent subagent task")
	}
	builder := a.client.AgentSubagentTask.Create().
		SetSessionID(task.SessionID).
		SetSubagentName(task.SubagentName).
		SetDescription(task.Description).
		SetPrompt(task.Prompt).
		SetStatus(task.Status).
		SetResult(task.Result).
		SetErrorText(task.ErrorText).
		SetDepth(task.Depth)
	if !task.StartedAt.IsZero() {
		builder = builder.SetStartedAt(task.StartedAt)
	}
	if !task.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(task.UpdatedAt)
	}
	if task.FinishedAt != nil {
		builder = builder.SetFinishedAt(*task.FinishedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent subagent task: %w", err)
	}
	task.ID = row.ID
	return nil
}

func (a *adapter) UpdateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error {
	if task == nil {
		return errors.New("postgres: nil agent subagent task")
	}
	if task.ID == 0 {
		return errors.New("postgres: agent subagent task id is required")
	}
	builder := a.client.AgentSubagentTask.UpdateOneID(task.ID).
		SetStatus(task.Status).
		SetResult(task.Result).
		SetErrorText(task.ErrorText).
		SetUpdatedAt(time.Now())
	if task.FinishedAt != nil {
		builder = builder.SetFinishedAt(*task.FinishedAt)
	}
	if _, err := builder.Save(ctx); err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: update agent subagent task: %w", err)
	}
	return nil
}

func (a *adapter) ListAgentSubagentTasks(ctx context.Context, sessionID string, limit int) ([]*gen.AgentSubagentTask, error) {
	query := a.client.AgentSubagentTask.Query().
		Order(gen.Desc(agentsubagenttask.FieldCreatedAt))
	if sessionID != "" {
		query = query.Where(agentsubagenttask.SessionIDEQ(sessionID))
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent subagent tasks: %w", err)
	}
	return rows, nil
}

func (a *adapter) GetAgentSubagentTask(ctx context.Context, id int64) (*gen.AgentSubagentTask, error) {
	row, err := a.client.AgentSubagentTask.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent task: %w", err)
	}
	return row, nil
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

func (a *adapter) ListTokens(ctx context.Context) ([]model.TokenItem, error) {
	rows, err := a.client.Parameter.Query().
		Order(gen.Desc(parameter.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tokens: %w", err)
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	result := make([]model.TokenItem, 0, len(rows))
	for _, r := range rows {
		paramsKV := types.KV(r.Params)
		if _, hasScopes := paramsKV["scopes"]; !hasScopes {
			continue
		}
		if r.ExpiredAt.Before(cutoff) {
			if _, hasUsed := paramsKV["last_used_at"]; !hasUsed {
				continue
			}
		}
		uidStr, _ := paramsKV.String("uid")
		var scopes []string
		if raw, ok := paramsKV["scopes"]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						scopes = append(scopes, s)
					}
				}
			case []string:
				scopes = v
			}
		}
		var lastUsedAt *time.Time
		if usedStr, ok := paramsKV.String("last_used_at"); ok && usedStr != "" {
			if t, err := time.Parse(time.RFC3339Nano, usedStr); err == nil {
				lastUsedAt = &t
			}
		}
		result = append(result, model.TokenItem{
			Token:      r.Flag,
			UID:        types.Uid(uidStr),
			Scopes:     scopes,
			CreatedAt:  r.CreatedAt,
			LastUsedAt: lastUsedAt,
			ExpiredAt:  r.ExpiredAt,
		})
	}
	return result, nil
}

func (a *adapter) CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	params := types.KV{
		"uid":    string(uid),
		"scopes": scopes,
	}
	now := time.Now()
	_, err = a.client.Parameter.Create().
		SetFlag(auth.HashToken(token)).
		SetParams(map[string]any(params)).
		SetExpiredAt(expiresAt).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	return token, nil
}

func (a *adapter) RevokeToken(ctx context.Context, flag string) error {
	n, err := a.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: revoke token: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
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
	copy(result, bots)
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
	copy(result, platforms)
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
	channelModel.ID = c.ID
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
	copy(result, channels)
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
	copy(result, agents)
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
// NotifyChannel CRUD
// ---------------------------------------------------------------------------

func (a *adapter) CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error) {
	ch, err := a.client.NotifyChannel.Create().
		SetName(name).
		SetProtocol(protocol).
		SetURI(uri).
		SetEnabled(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify channel: %w", err)
	}
	return ch.ID, nil
}

func (a *adapter) GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := a.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel: %w", err)
	}
	return model.NotifyChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Protocol:  ch.Protocol,
		URI:       a.MaskNotifyURI(ch.Protocol, ch.URI),
		Enabled:   ch.Enabled,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}, nil
}

func (a *adapter) GetNotifyChannelRaw(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := a.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel raw: %w", err)
	}
	return model.NotifyChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Protocol:  ch.Protocol,
		URI:       ch.URI,
		Enabled:   ch.Enabled,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}, nil
}

func (a *adapter) ListNotifyChannels(ctx context.Context, opts store.ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	q := a.client.NotifyChannel.Query()
	if opts.Protocol != "" {
		q = q.Where(notifychannel.Protocol(opts.Protocol))
	}
	if opts.Enabled != nil {
		q = q.Where(notifychannel.Enabled(*opts.Enabled))
	}
	chs, err := q.Order(gen.Asc(notifychannel.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify channels: %w", err)
	}
	result := make([]model.NotifyChannel, len(chs))
	for i, ch := range chs {
		result[i] = model.NotifyChannel{
			ID:        ch.ID,
			Name:      ch.Name,
			Protocol:  ch.Protocol,
			URI:       a.MaskNotifyURI(ch.Protocol, ch.URI),
			Enabled:   ch.Enabled,
			CreatedAt: ch.CreatedAt,
			UpdatedAt: ch.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) UpdateNotifyChannel(ctx context.Context, id int64, name, protocol, uri string, enabled bool) error {
	upd := a.client.NotifyChannel.Update().Where(notifychannel.IDEQ(id)).
		SetName(name).
		SetProtocol(protocol).
		SetEnabled(enabled).
		SetUpdatedAt(time.Now())
	if uri != "" {
		upd = upd.SetURI(uri)
	}
	n, err := upd.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify channel: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteNotifyChannel(ctx context.Context, id int64) error {
	_, err := a.client.NotifyChannel.Delete().Where(notifychannel.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify channel: %w", err)
	}
	return nil
}

// MaskNotifyURI produces a display-safe masked form of a notification URI.
func (a *adapter) MaskNotifyURI(protocol, uri string) string {
	_ = a
	switch protocol {
	case "slack":
		return maskSlackURI(uri)
	case "ntfy":
		return maskNtfyURI(uri)
	case "pushover":
		return maskPushoverURI(uri)
	case "message-pusher":
		return maskMessagePusherURI(uri)
	default:
		if len(uri) > 30 {
			return uri[:27] + "..."
		}
		return uri
	}
}

func maskSlackURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "slack://******"
	}
	pathParts := strings.Split(parts[1], "/")
	if len(pathParts) > 3 {
		pathParts[len(pathParts)-3] = "T******"
	}
	if len(pathParts) > 2 {
		pathParts[len(pathParts)-2] = "B******"
	}
	if len(pathParts) > 1 {
		pathParts[len(pathParts)-1] = "C******"
	}
	return parts[0] + "://" + strings.Join(pathParts, "/")
}

func maskNtfyURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "ntfy://******"
	}
	hostParts := strings.SplitN(parts[1], "/", 2)
	if len(hostParts) < 2 {
		return parts[0] + "://" + hostParts[0] + "/******"
	}
	return parts[0] + "://" + hostParts[0] + "/******"
}

func maskPushoverURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "pushover://******"
	}
	userIdx := strings.Index(parts[1], "@")
	if userIdx < 0 {
		return parts[0] + "://U******@" + maskEnd(parts[1])
	}
	return parts[0] + "://U******@A******"
}

func maskMessagePusherURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "message-pusher://******"
	}
	atIdx := strings.Index(parts[1], "@")
	if atIdx < 0 {
		return parts[0] + "://******"
	}
	finalSlash := strings.LastIndex(parts[1], "/")
	if finalSlash < 0 {
		return parts[0] + "://" + parts[1][:atIdx+1] + "domain/******/******"
	}
	secondLast := strings.LastIndex(parts[1][:finalSlash], "/")
	if secondLast < 0 {
		return parts[0] + "://" + parts[1][:finalSlash+1] + "******"
	}
	return parts[0] + "://" + parts[1][:secondLast+1] + "******/******"
}

func maskEnd(s string) string {
	if len(s) > 8 {
		return s[:4] + "******"
	}
	return "******"
}

// ---------------------------------------------------------------------------
// NotifyRule CRUD
// ---------------------------------------------------------------------------

func (a *adapter) CreateNotifyRule(ctx context.Context, rule model.NotifyRule) (int64, error) {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return 0, fmt.Errorf("postgres: create notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	r, err := a.client.NotifyRule.Create().
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify rule: %w", err)
	}
	return r.ID, nil
}

func (a *adapter) GetNotifyRule(ctx context.Context, id int64) (model.NotifyRule, error) {
	r, err := a.client.NotifyRule.Query().Where(notifyrule.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyRule{}, types.ErrNotFound
		}
		return model.NotifyRule{}, fmt.Errorf("postgres: get notify rule: %w", err)
	}
	return notifyRuleToModel(r), nil
}

func (a *adapter) ListNotifyRules(ctx context.Context, opts store.ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	q := a.client.NotifyRule.Query()
	if opts.Enabled != nil {
		q = q.Where(notifyrule.Enabled(*opts.Enabled))
	}
	rules, err := q.Order(gen.Desc(notifyrule.FieldPriority)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify rules: %w", err)
	}
	result := make([]model.NotifyRule, len(rules))
	for i, r := range rules {
		result[i] = notifyRuleToModel(r)
	}
	return result, nil
}

func (a *adapter) UpdateNotifyRule(ctx context.Context, id int64, rule model.NotifyRule) error {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return fmt.Errorf("postgres: update notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	n, err := a.client.NotifyRule.Update().Where(notifyrule.IDEQ(id)).
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify rule: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteNotifyRule(ctx context.Context, id int64) error {
	_, err := a.client.NotifyRule.Delete().Where(notifyrule.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify rule: %w", err)
	}
	return nil
}

func notifyRuleToModel(r *gen.NotifyRule) model.NotifyRule {
	paramsJSON, _ := sonic.MarshalString(r.Params)
	return model.NotifyRule{
		ID:             r.ID,
		RuleID:         r.RuleID,
		Name:           r.Name,
		Action:         string(r.Action),
		EventPattern:   r.EventPattern,
		ChannelPattern: r.ChannelPattern,
		Condition:      r.Condition,
		Priority:       r.Priority,
		ParamsJSON:     paramsJSON,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

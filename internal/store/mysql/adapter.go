package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/dao"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	ms "github.com/go-sql-driver/mysql"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	adapterName = "mysql"

	defaultDatabase = "flowbot"

	defaultMaxResults = 1024
	// This is capped by the Session's send queue limit (128).
	defaultMaxMessageResults = 100

	// If DB request timeout is specified,
	// we allocate txTimeoutMultiplier times more time for transactions.
	txTimeoutMultiplier = 1.5
)

type configType struct {
	// DB connection settings.
	// Please, see https://pkg.go.dev/github.com/go-sql-driver/mysql#Config
	// for the full list of fields.
	ms.Config
	// mysql DSN
	DSN string `json:"dsn,omitempty"`

	// Connection pool settings.
	//
	// Maximum number of open connections to the database.
	MaxOpenConns int `json:"max_open_conns,omitempty"`
	// Maximum number of connections in the idle connection pool.
	MaxIdleConns int `json:"max_idle_conns,omitempty"`
	// Maximum amount of time a connection may be reused (in seconds).
	ConnMaxLifetime int `json:"conn_max_lifetime,omitempty"`

	// DB request timeout (in seconds).
	// If 0 (or negative), no timeout is applied.
	SqlTimeout int `json:"sql_timeout,omitempty"`
}

func Init() {
	store.RegisterAdapter(&adapter{})
}

type adapter struct {
	db *gorm.DB

	dsn    string
	dbName string
	// Maximum number of records to return
	maxResults int
	// Maximum number of message records to return
	maxMessageResults int

	// Single query timeout.
	sqlTimeout time.Duration
	// DB transaction timeout.
	txTimeout time.Duration
}

func (a *adapter) SetMaxResults(val int) error {
	a.maxResults = val
	return nil
}

func (a *adapter) UserCreate(user *model.User) error {
	q := dao.Q.User
	return q.Create(user)
}

func (a *adapter) UserGet(uid types.Uid) (*model.User, error) {
	q := dao.Q.User
	return q.
		Where(q.Flag.Eq(uid.String())).
		First()
}

func (a *adapter) UserGetAll(uid ...types.Uid) ([]*model.User, error) {
	q := dao.Q.User
	if len(uid) > 0 {
		s := make([]string, len(uid))
		for _, u := range uid {
			s = append(s, u.String())
		}
		q.Where(q.Flag.In(s...))
	}
	return q.Find()
}

func (a *adapter) FirstUser() (*model.User, error) {
	q := dao.Q.User
	return q.First()
}

func (a *adapter) UserDelete(uid types.Uid, _ bool) error {
	q := dao.Q.User
	_, err := q.
		Where(q.Flag.Eq(uid.String())).
		Delete()
	return err
}

func (a *adapter) UserUpdate(uid types.Uid, update types.KV) error {
	q := dao.Q.User
	_, err := q.
		Where(q.Flag.Eq(uid.String())).
		UpdateColumns(update)
	return err
}

func (a *adapter) FileStartUpload(fd *types.FileDef) error {
	q := dao.Q.Fileupload
	return q.Create(&model.Fileupload{
		UID:      fd.User,
		Fid:      fd.Id,
		Name:     fd.Name,
		Mimetype: fd.MimeType,
		Location: fd.Location,
		State:    model.FileStart,
	})
}

func (a *adapter) FileFinishUpload(fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	q := dao.Q.Fileupload
	res, err := q.Where(q.Fid.Eq(fd.Id)).First()
	if err != nil {
		return nil, err
	}
	if success {
		res.State = model.FileFinish
		res.Size = size
	} else {
		res.State = model.FileFailed
	}
	_, err = q.
		Where(q.Fid.Eq(fd.Id)).
		Updates(res)
	if err != nil {
		return nil, err
	}
	if success {
		return fd, nil
	}
	return nil, nil
}

func (a *adapter) FileGet(fid string) (*types.FileDef, error) {
	q := dao.Q.Fileupload
	res, err := q.
		Where(q.Fid.Eq(fid)).
		First()
	if err != nil {
		return nil, err
	}
	return &types.FileDef{
		ObjHeader: types.ObjHeader{
			Id:        fid,
			CreatedAt: res.CreatedAt,
			UpdatedAt: res.UpdatedAt,
		},
		User:     res.UID,
		Name:     res.Name,
		Status:   int(res.State),
		MimeType: res.Mimetype,
		Size:     res.Size,
		Location: res.Location,
	}, nil
}

func (a *adapter) FileDeleteUnused(olderThan time.Time, limit int) ([]string, error) {
	q := dao.Q.Fileupload
	// Find files that are not in FileFinish state and older than the given time.
	list, err := q.
		Where(q.State.Neq(model.FileFinish), q.CreatedAt.Lt(olderThan)).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}

	// Collect locations for actual file deletion.
	result := make([]string, 0, len(list))
	fids := make([]string, 0, len(list))
	for _, f := range list {
		if f.Location != "" {
			result = append(result, f.Location)
		}
		fids = append(fids, f.Fid)
	}

	// Delete the database records.
	if len(fids) > 0 {
		_, err = q.Where(q.Fid.In(fids...)).Delete()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func isMissingDb(err error) bool {
	if err == nil {
		return false
	}

	var myErr *ms.MySQLError
	ok := errors.As(err, &myErr)
	return ok && myErr.Number == 1049
}

func (a *adapter) Open(adaptersConfig config.StoreType) error {
	if a.db != nil {
		return errors.New("mysql adapter is already connected")
	}

	if adaptersConfig.UseAdapter == "" {
		return errors.New("adapter name missing config")
	}
	if adaptersConfig.UseAdapter != "mysql" {
		return errors.New("adapter name must be 'mysql'")
	}

	var err error
	defaultCfg := ms.NewConfig()
	conf := configType{Config: *defaultCfg}

	data, err := sonic.Marshal(adaptersConfig.Adapters["mysql"])
	if err != nil {
		return errors.New("mysql adapter failed to parse config: " + err.Error())
	}
	if err = sonic.Unmarshal(data, &conf); err != nil {
		return errors.New("mysql adapter failed to parse config: " + err.Error())
	}

	if a.dsn == "" {
		a.dsn = conf.DSN
	}

	if a.dbName == "" {
		a.dbName = defaultDatabase
	}

	if a.maxResults <= 0 {
		a.maxResults = defaultMaxResults
	}

	if a.maxMessageResults <= 0 {
		a.maxMessageResults = defaultMaxMessageResults
	}

	// This just initializes the driver but does not open the network connection.
	db, err := sql.Open("mysql", a.dsn)
	if err != nil {
		return err
	}

	// Actually opening the network connection.
	err = db.Ping()
	if isMissingDb(err) {
		// Ignore missing database here. If we are initializing the database
		// missing DB is OK.
		err = nil
	}
	if err == nil {
		if conf.MaxOpenConns > 0 {
			db.SetMaxOpenConns(conf.MaxOpenConns)
		}
		if conf.MaxIdleConns > 0 {
			db.SetMaxIdleConns(conf.MaxIdleConns)
		}
		if conf.ConnMaxLifetime > 0 {
			db.SetConnMaxLifetime(time.Duration(conf.ConnMaxLifetime) * time.Second)
		}
		if conf.SqlTimeout > 0 {
			a.sqlTimeout = time.Duration(conf.SqlTimeout) * time.Second
			// We allocate txTimeoutMultiplier times sqlTimeout for transactions.
			a.txTimeout = time.Duration(float64(conf.SqlTimeout)*txTimeoutMultiplier) * time.Second
		}
	}

	a.db, err = gorm.Open(mysqlDriver.New(mysqlDriver.Config{Conn: db}), &gorm.Config{
		Logger:                 flog.NewGormLogger(config.App.Log.Level),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}

	// Initialize dao
	dao.SetDefault(a.db)

	return nil
}

func (a *adapter) IsOpen() bool {
	return a.db != nil
}

func (a *adapter) Close() error {
	rawDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return rawDB.Close()
}

func (a *adapter) GetName() string {
	return adapterName
}

func (a *adapter) Stats() any {
	if a.db == nil {
		return nil
	}
	rawDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return rawDB.Stats()
}

func (a *adapter) GetDB() *gorm.DB {
	return a.db
}

func (a *adapter) GetUsers() ([]*model.User, error) {
	q := dao.Q.User
	return q.Find()
}

func (a *adapter) GetUserById(id int64) (*model.User, error) {
	q := dao.Q.User
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetUserByFlag(flag string) (*model.User, error) {
	q := dao.Q.User
	return q.
		Where(q.Flag.Eq(flag)).
		First()
}

func (a *adapter) CreatePlatformUser(item *model.PlatformUser) (int64, error) {
	q := dao.Q.PlatformUser
	err := q.
		Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) GetPlatformUserByFlag(flag string) (*model.PlatformUser, error) {
	q := dao.Q.PlatformUser
	return q.
		Where(q.Flag.Eq(flag)).
		First()
}

func (a *adapter) GetPlatformUsersByUserId(userId int64) ([]*model.PlatformUser, error) {
	q := dao.Q.PlatformUser
	return q.
		Where(q.UserID.Eq(userId)).
		Find()
}

func (a *adapter) UpdatePlatformUser(item *model.PlatformUser) error {
	q := dao.Q.PlatformUser
	_, err := q.Updates(item)
	return err
}

func (a *adapter) GetPlatformChannelByFlag(flag string) (*model.PlatformChannel, error) {
	q := dao.Q.PlatformChannel
	return q.
		Where(q.Flag.Eq(flag)).
		First()
}

func (a *adapter) GetPlatformChannelsByPlatformIds(platformIds []int64) ([]*model.PlatformChannel, error) {
	q := dao.Q.PlatformChannel
	return q.
		Where(q.PlatformID.In(platformIds...)).
		Find()
}

func (a *adapter) GetPlatformChannelsByChannelId(channelId int64) (*model.PlatformChannel, error) {
	q := dao.Q.PlatformChannel
	return q.
		Where(q.ChannelID.Eq(channelId)).
		First()
}

func (a *adapter) CreatePlatformChannel(item *model.PlatformChannel) (int64, error) {
	q := dao.Q.PlatformChannel
	err := q.
		Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) CreatePlatformChannelUser(item *model.PlatformChannelUser) (int64, error) {
	q := dao.Q.PlatformChannelUser
	err := q.
		Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) GetPlatformChannelUsersByUserFlag(userFlag string) ([]*model.PlatformChannelUser, error) {
	q := dao.Q.PlatformChannelUser
	return q.
		Where(q.UserFlag.Eq(userFlag)).
		Find()
}

func (a *adapter) GetMessage(flag string) (*model.Message, error) {
	q := dao.Q.Message
	return q.
		Where(q.Flag.Eq(flag)).
		First()
}

func (a *adapter) GetMessageByPlatform(platformId int64, platformMsgId string) (*model.Message, error) {
	q := dao.Q.Message
	return q.
		Where(q.PlatformID.Eq(platformId), q.PlatformMsgID.Eq(platformMsgId)).
		First()
}

func (a *adapter) GetMessagesBySession(session string) ([]*model.Message, error) {
	q := dao.Q.Message
	return q.
		Where(q.Session.Eq(session)).
		Order(q.CreatedAt.Asc()).
		Find()
}

func (a *adapter) CreateMessage(message model.Message) error {
	q := dao.Q.Message
	return q.
		Create(&message)
}

func (a *adapter) DataSet(uid types.Uid, topic, key string, value types.KV) error {
	var find model.Data
	err := a.db.Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.Data{}).
			Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).
			Update("value", value).Error
	} else {
		return a.db.Create(&model.Data{
			UID:   string(uid),
			Topic: topic,
			Key:   key,
			Value: model.JSON(value),
		}).Error
	}
}

func (a *adapter) DataGet(uid types.Uid, topic, key string) (types.KV, error) {
	var find model.Data
	err := a.db.Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).First(&find).Error
	if err != nil {
		return nil, err
	}
	return types.KV(find.Value), nil
}

func (a *adapter) DataList(uid types.Uid, topic string, filter types.DataFilter) ([]*model.Data, error) {
	var list []*model.Data
	builder := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic)
	if filter.Prefix != nil {
		builder = builder.
			Where("`key` LIKE ?", fmt.Sprintf("%s%%", *filter.Prefix))
	}
	if filter.CreatedStart != nil {
		builder = builder.
			Where("created_at >= ?", filter.CreatedStart)
	}
	if filter.CreatedEnd != nil {
		builder = builder.
			Where("created_at <= ?", filter.CreatedEnd)
	}
	err := builder.
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (a *adapter) DataDelete(uid types.Uid, topic string, key string) error {
	return a.db.
		Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).
		Delete(&model.Data{}).Error
}

func (a *adapter) ConfigSet(uid types.Uid, topic, key string, value types.KV) error {
	var find model.Config
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).
		First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.Config{}).
			Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).
			Update("value", value).Error
	} else {
		return a.db.Create(&model.Config{
			UID:   string(uid),
			Topic: topic,
			Key:   key,
			Value: model.JSON(value),
		}).Error
	}
}

func (a *adapter) ConfigGet(uid types.Uid, topic, key string) (types.KV, error) {
	var find model.Config
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND `key` = ?", uid, topic, key).
		First(&find).Error
	if err != nil {
		return nil, err
	}
	return types.KV(find.Value), nil
}

func (a *adapter) ListConfigByPrefix(uid types.Uid, topic string, prefix string) ([]*model.Config, error) {
	q := dao.Q.Config
	return q.Where(q.UID.Eq(uid.String()), q.Topic.Eq(topic), q.Key.Like(fmt.Sprintf("%s%%", prefix))).
		Find()
}

func (a *adapter) ConfigDelete(uid types.Uid, topic string, key string) error {
	q := dao.Q.Config
	_, err := q.Where(q.UID.Eq(uid.String()), q.Topic.Eq(topic), q.Key.Eq(key)).
		Delete()
	return err
}

func (a *adapter) OAuthSet(oauth model.OAuth) error {
	var find model.OAuth
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND `type` = ?", oauth.UID, oauth.Topic, oauth.Type).
		First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.OAuth{}).
			Where("`uid` = ? AND `topic` = ? AND `type` = ?", oauth.UID, oauth.Topic, oauth.Type).
			UpdateColumns(types.KV{
				"token": oauth.Token,
				"extra": oauth.Extra,
			}).Error
	} else {
		return a.db.Create(&oauth).Error
	}
}

func (a *adapter) OAuthGet(uid types.Uid, topic, t string) (model.OAuth, error) {
	var find model.OAuth
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND `type` = ?", uid, topic, t).
		First(&find).Error
	if err != nil {
		return model.OAuth{}, err
	}
	return find, nil
}

func (a *adapter) OAuthGetAvailable(t string) ([]model.OAuth, error) {
	var find []model.OAuth
	err := a.db.
		Where("`type` = ? AND `token` <> ''", t).
		Find(&find).Error
	if err != nil {
		return []model.OAuth{}, err
	}
	return find, nil
}

func (a *adapter) FormSet(formId string, form model.Form) error {
	var find model.Form
	err := a.db.
		Where("`form_id` = ?", formId).
		First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.Form{}).
			Where("`form_id` = ?", formId).
			UpdateColumns(map[string]any{
				"values":     form.Values,
				"state":      form.State,
				"updated_at": time.Now(),
			}).Error
	} else {
		return a.db.Create(&model.Form{
			FormID: formId,
			UID:    form.UID,
			Topic:  form.Topic,
			Schema: form.Schema,
			Values: form.Values,
			State:  form.State,
			Extra:  form.Extra,
		}).Error
	}
}

func (a *adapter) FormGet(formId string) (model.Form, error) {
	var find model.Form
	err := a.db.
		Where("`form_id` = ?", formId).
		First(&find).Error
	if err != nil {
		return model.Form{}, err
	}
	return find, nil
}

func (a *adapter) BehaviorSet(behavior model.Behavior) error {
	return a.db.Create(&behavior).Error
}

func (a *adapter) BehaviorGet(uid types.Uid, flag string) (model.Behavior, error) {
	var find model.Behavior
	err := a.db.
		Where("`uid` = ? AND `flag` = ?", uid, flag).
		First(&find).Error
	if err != nil {
		return model.Behavior{}, err
	}
	return find, nil
}

func (a *adapter) BehaviorList(uid types.Uid) ([]*model.Behavior, error) {
	var list []*model.Behavior
	err := a.db.
		Where("`uid` = ?", uid).Order("id DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (a *adapter) BehaviorIncrease(uid types.Uid, flag string, number int) error {
	return a.db.
		Model(&model.Behavior{}).
		Where("`uid` = ? AND `flag` = ?", uid, flag).
		UpdateColumn("count", gorm.Expr("count + ?", number)).Error
}

func (a *adapter) ParameterSet(flag string, params types.KV, expiredAt time.Time) error {
	var find model.Parameter
	err := a.db.Where("`flag` = ?", flag).First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.Parameter{}).
			Where("`flag` = ?", flag).
			Update("expired_at", expiredAt).Error
	} else {
		return a.db.Create(&model.Parameter{
			Flag:      flag,
			Params:    model.JSON(params),
			ExpiredAt: expiredAt,
		}).Error
	}
}

func (a *adapter) ParameterGet(flag string) (model.Parameter, error) {
	var find model.Parameter
	err := a.db.
		Where("`flag` = ?", flag).
		First(&find).Error
	if err != nil {
		return model.Parameter{}, err
	}
	return find, nil
}

func (a *adapter) ParameterDelete(flag string) error {
	_, err := dao.Q.Parameter.
		Where(dao.Parameter.Flag.Eq(flag)).
		Delete()
	return err
}

func (a *adapter) PageSet(pageId string, page model.Page) error {
	var find model.Page
	err := a.db.
		Where("`page_id` = ?", pageId).
		First(&find).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if find.ID > 0 {
		return a.db.
			Model(&model.Page{}).
			Where("`page_id` = ?", pageId).
			Update("state", int(page.State)).
			Error
	} else {
		return a.db.Create(&page).Error
	}
}

func (a *adapter) PageGet(pageId string) (model.Page, error) {
	var find model.Page
	err := a.db.
		Where("`page_id` = ?", pageId).
		First(&find).Error
	if err != nil {
		return model.Page{}, err
	}
	return find, nil
}

func (a *adapter) CreateInstruct(instruct *model.Instruct) (int64, error) {
	if instruct.ExpireAt.Before(time.Now()) {
		return 0, errors.New("expire time error")
	}
	err := a.db.
		Create(&instruct)
	if err != nil {
		return 0, nil
	}
	return instruct.ID, nil
}

func (a *adapter) ListInstruct(uid types.Uid, isExpire bool, limit int) ([]*model.Instruct, error) {
	var items []*model.Instruct
	builder := a.db.
		Where("`uid` = ?", uid).
		Where("state = ?", model.InstructCreate)
	if isExpire {
		builder.Where("expire_at < ?", time.Now())
	} else {
		builder.Where("expire_at >= ?", time.Now())
	}

	err := builder.
		Order("priority DESC").
		Order("updated_at DESC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (a *adapter) UpdateInstruct(instruct *model.Instruct) error {
	return a.db.
		Model(&model.Instruct{}).
		Where("`no` = ?", instruct.No).
		Update("state", instruct.State).
		Error
}

func (a *adapter) ListWebhook(uid types.Uid) ([]*model.Webhook, error) {
	q := dao.Q.Webhook
	return q.
		Where(q.UID.Eq(uid.String())).
		Find()
}

func (a *adapter) CreateWebhook(webhook *model.Webhook) (int64, error) {
	q := dao.Q.Webhook
	err := q.Create(webhook)
	if err != nil {
		return 0, err
	}
	return webhook.ID, nil
}

func (a *adapter) UpdateWebhook(webhook *model.Webhook) error {
	q := dao.Q.Webhook
	_, err := q.
		Where(q.ID.Eq(webhook.ID)).
		Update(q.State, webhook.State)
	return err
}

func (a *adapter) DeleteWebhook(id int64) error {
	q := dao.Q.Webhook
	_, err := q.Where(q.ID.Eq(id)).Delete()
	return err
}

func (a *adapter) IncreaseWebhookCount(id int64) error {
	q := dao.Q.Webhook
	_, err := q.
		Where(q.ID.Eq(id)).
		Update(q.TriggerCount, gorm.Expr("trigger_count + ?", 1))
	return err
}

func (a *adapter) GetWebhookBySecret(secret string) (*model.Webhook, error) {
	q := dao.Q.Webhook
	return q.
		Where(q.Secret.Eq(secret)).
		First()
}

func (a *adapter) GetWebhookByUidAndFlag(uid types.Uid, flag string) (*model.Webhook, error) {
	q := dao.Q.Webhook
	return q.
		Where(q.UID.Eq(uid.String())).
		Where(q.Flag.Eq(flag)).
		First()
}

func (a *adapter) CreateCounter(counter *model.Counter) (int64, error) {
	err := a.db.
		Create(&counter)
	if err != nil {
		return 0, nil
	}
	a.record(counter.ID, counter.Digit)
	return counter.ID, nil
}

func (a *adapter) IncreaseCounter(id, amount int64) error {
	err := a.db.
		Model(&model.Counter{}).
		Where("id = ?", id).
		Update("digit", gorm.Expr("digit + ?", amount)).Error
	if err != nil {
		return err
	}
	a.record(id, amount)
	return nil
}

func (a *adapter) DecreaseCounter(id, amount int64) error {
	err := a.db.
		Model(&model.Counter{}).
		Where("id = ?", id).
		Update("digit", gorm.Expr("digit - ?", amount)).Error
	if err != nil {
		return err
	}
	a.record(id, -amount)
	return nil
}

func (a *adapter) ListCounter(uid types.Uid, topic string) ([]*model.Counter, error) {
	var items []*model.Counter
	err := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic).
		Order("updated_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (a *adapter) record(id, digit int64) {
	q := dao.Q.CounterRecord
	err := q.
		Create(&model.CounterRecord{
			CounterID: id,
			Digit:     int32(digit),
		})
	if err != nil {
		flog.Error(err)
	}
}

func (a *adapter) GetCounter(id int64) (model.Counter, error) {
	var find model.Counter
	err := a.db.
		Where("id = ?", id).
		First(&find).Error
	if err != nil {
		return model.Counter{}, err
	}
	return find, nil
}

func (a *adapter) GetCounterByFlag(uid types.Uid, topic string, flag string) (model.Counter, error) {
	var find model.Counter
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND flag = ?", uid, topic, flag).
		First(&find).Error
	if err != nil {
		return model.Counter{}, err
	}
	return find, nil
}

func (a *adapter) GetBot(id int64) (*model.Bot, error) {
	q := dao.Q.Bot
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetBotByName(name string) (*model.Bot, error) {
	q := dao.Q.Bot
	return q.
		Where(q.Name.Eq(name)).
		First()
}

func (a *adapter) CreateBot(bot *model.Bot) (int64, error) {
	q := dao.Q.Bot
	err := q.Create(bot)
	if err != nil {
		return 0, err
	}
	return bot.ID, nil
}

func (a *adapter) UpdateBot(bot *model.Bot) error {
	q := dao.Q.Bot
	_, err := q.
		Where(q.Name.Eq(bot.Name)).
		Update(q.State, bot.State)
	return err
}

func (a *adapter) DeleteBot(name string) error {
	q := dao.Q.Bot
	_, err := q.
		Where(q.Name.Eq(name)).
		Delete()
	return err
}

func (a *adapter) GetBots() ([]*model.Bot, error) {
	q := dao.Q.Bot
	return q.Find()
}

func (a *adapter) GetPlatform(id int64) (*model.Platform, error) {
	q := dao.Q.Platform
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetPlatformByName(name string) (*model.Platform, error) {
	q := dao.Q.Platform
	return q.
		Where(q.Name.Eq(name)).
		First()
}

func (a *adapter) GetPlatforms() ([]*model.Platform, error) {
	q := dao.Q.Platform
	return q.Find()
}

func (a *adapter) CreatePlatform(platform *model.Platform) (int64, error) {
	q := dao.Q.Platform
	err := q.Create(platform)
	if err != nil {
		return 0, err
	}
	return platform.ID, nil
}

func (a *adapter) GetChannel(id int64) (*model.Channel, error) {
	q := dao.Q.Channel
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetChannelByName(name string) (*model.Channel, error) {
	q := dao.Q.Channel
	return q.
		Where(q.Name.Eq(name)).
		First()
}

func (a *adapter) CreateChannel(channel *model.Channel) (int64, error) {
	q := dao.Q.Channel
	err := q.Create(channel)
	if err != nil {
		return 0, err
	}
	return channel.ID, nil
}

func (a *adapter) UpdateChannel(channel *model.Channel) error {
	q := dao.Q.Channel
	_, err := q.
		Where(q.Name.Eq(channel.Name)).
		Update(q.State, channel.State)
	return err
}

func (a *adapter) DeleteChannel(name string) error {
	q := dao.Q.Channel
	_, err := q.
		Where(q.Name.Eq(name)).
		Delete()
	return err
}

func (a *adapter) GetChannels() ([]*model.Channel, error) {
	q := dao.Q.Channel
	return q.Find()
}

func (a *adapter) GetAgents() ([]*model.Agent, error) {
	q := dao.Q.Agent
	return q.Find()
}

func (a *adapter) GetAgentByHostid(uid types.Uid, topic string, hostid string) (*model.Agent, error) {
	q := dao.Q.Agent
	return q.
		Where(q.UID.Eq(uid.String())).
		Where(q.Topic.Eq(topic)).
		Where(q.Hostid.Eq(hostid)).
		First()
}

func (a *adapter) CreateAgent(agent *model.Agent) (int64, error) {
	q := dao.Q.Agent
	err := q.Create(agent)
	if err != nil {
		return 0, err
	}
	return agent.ID, nil
}

func (a *adapter) UpdateAgentLastOnlineAt(uid types.Uid, topic string, hostid string, lastOnlineAt time.Time) error {
	q := dao.Q.Agent
	_, err := q.
		Where(q.UID.Eq(uid.String())).
		Where(q.Topic.Eq(topic)).
		Where(q.Hostid.Eq(hostid)).
		UpdateColumn(q.LastOnlineAt, lastOnlineAt)
	return err
}

func (a *adapter) UpdateAgentOnlineDuration(uid types.Uid, topic string, hostid string, offlineTime time.Time) error {
	agent, err := a.GetAgentByHostid(uid, topic, hostid)
	if err != nil {
		return err
	}
	if agent.LastOnlineAt.IsZero() {
		return nil
	}

	duration := offlineTime.Sub(agent.LastOnlineAt)
	if duration < 0 {
		return nil
	}

	q := dao.Q.Agent
	_, err = q.
		Where(q.UID.Eq(uid.String())).
		Where(q.Topic.Eq(topic)).
		Where(q.Hostid.Eq(hostid)).
		UpdateColumn(q.OnlineDuration, agent.OnlineDuration+int32(duration.Seconds()))
	return err
}

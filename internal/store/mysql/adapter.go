package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/dao"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/locker"
	ms "github.com/go-sql-driver/mysql"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gen/field"
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

func (a *adapter) UserDelete(uid types.Uid, _ bool) error {
	q := dao.Q.User
	_, err := q.
		Where(q.Flag.Eq(uid.String())).
		Delete()
	return err
}

func (a *adapter) UserUpdate(uid types.Uid, update map[string]interface{}) error {
	q := dao.Q.User
	_, err := q.
		Where(q.Flag.Eq(uid.String())).
		UpdateColumns(update)
	return err
}

func (a *adapter) FileStartUpload(fd *types.FileDef) error {
	q := dao.Q.Fileupload
	return q.Create(&model.Fileupload{
		UID:      fd.Id,
		Mimetype: fd.MimeType,
		Location: fd.Location,
		State:    model.FileStart,
	})
}

func (a *adapter) FileFinishUpload(fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	q := dao.Q.Fileupload
	res, err := q.Where(q.UID.Eq(fd.Id)).First()
	if err != nil {
		return nil, err
	}
	if success {
		res.State = model.FileFinish
		res.Size = size
		_, err = q.
			Where(q.UID.Eq(fd.Id)).
			Updates(res)
		return fd, err
	} else {
		_, err = q.
			Where(q.UID.Eq(fd.Id)).
			Delete()
		return nil, err
	}
}

func (a *adapter) FileGet(fid string) (*types.FileDef, error) {
	q := dao.Q.Fileupload
	res, err := q.
		Where(q.UID.Eq(fid)).
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
		Status:   int(res.State),
		MimeType: res.Mimetype,
		Size:     res.Size,
		Location: res.Location,
	}, nil
}

func (a *adapter) FileDeleteUnused(olderThan time.Time, limit int) ([]string, error) {
	q := dao.Q.Fileupload
	list, err := q.
		Where(q.CreatedAt.Lt(olderThan)).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	result := make([]string, len(list))
	for i, f := range list {
		result[i] = f.UID
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

	data, err := json.Marshal(adaptersConfig.Adapters["mysql"])
	if err != nil {
		return errors.New("mysql adapter failed to parse config: " + err.Error())
	}
	if err = json.Unmarshal(data, &conf); err != nil {
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
		Logger: flog.NewGormLogger(config.App.Log.Level),
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

func (a *adapter) Stats() interface{} {
	if a.db == nil {
		return nil
	}
	rawDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return rawDB.Stats()
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

func (a *adapter) CreatePlatformChannel(item *model.PlatformChannel) (int64, error) {
	q := dao.Q.PlatformChannel
	err := q.
		Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
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
			UpdateColumns(map[string]interface{}{
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
			Updates(map[string]interface{}{
				"values": form.Values,
				"state":  form.State,
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
			Updates(map[string]interface{}{
				"state": page.State,
			}).Error
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

func (a *adapter) ListInstruct(uid types.Uid, isExpire bool) ([]*model.Instruct, error) {
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
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (a *adapter) UpdateInstruct(instruct *model.Instruct) error {
	return a.db.
		Model(&model.Todo{}).
		Where("`no` = ?", instruct.No).
		UpdateColumns(map[string]interface{}{
			"state": instruct.State,
		}).Error
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

func (a *adapter) GetObjectiveByID(id int64) (*model.Objective, error) {
	var objective model.Objective
	err := a.db.
		Where("id = ?", id).
		First(&objective).Error
	if err != nil {
		return nil, err
	}
	return &objective, nil
}

func (a *adapter) GetObjectiveBySequence(uid types.Uid, topic string, sequence int64) (*model.Objective, error) {
	var objective model.Objective
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		Preload("KeyResults").
		First(&objective).Error
	if err != nil {
		return nil, err
	}
	return &objective, nil
}

func (a *adapter) ListObjectives(uid types.Uid, topic string) ([]*model.Objective, error) {
	var objectives []*model.Objective
	err := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic).
		Order("id DESC").
		Find(&objectives).Error
	if err != nil {
		return nil, err
	}
	return objectives, nil
}

func (a *adapter) CreateObjective(objective *model.Objective) (int64, error) {
	ctx := context.Background()
	l := locker.NewLocker()
	lock, err := l.Acquire(ctx, fmt.Sprintf("chatbot:objective:create:%s", objective.UID), 10*time.Second)
	if err != nil {
		return 0, err
	}
	defer func(lock *redislock.Lock, ctx context.Context) {
		_ = lock.Release(ctx)
	}(lock, ctx)

	// sequence
	sequence := int32(0)
	var maxObjective model.Objective
	err = a.db.
		Where("`uid` = ? AND `topic` = ?", objective.UID, objective.Topic).
		Order("sequence DESC").
		Take(&maxObjective).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if maxObjective.Sequence > 0 {
		sequence = maxObjective.Sequence
	}
	sequence += 1

	if objective.CreatedData.IsZero() {
		objective.CreatedData = time.Now()
	}
	if objective.UpdatedDate.IsZero() {
		objective.UpdatedDate = time.Now()
	}
	objective.Sequence = sequence
	err = a.db.Create(&objective).Error
	if err != nil {
		return 0, err
	}
	return objective.ID, nil
}

func (a *adapter) UpdateObjective(objective *model.Objective) error {
	return a.db.Model(&model.Objective{}).
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", objective.UID, objective.Topic, objective.Sequence).
		UpdateColumns(map[string]interface{}{
			"title":       objective.Title,
			"memo":        objective.Memo,
			"motive":      objective.Motive,
			"feasibility": objective.Feasibility,
			"is_plan":     objective.IsPlan,
			"plan_start":  objective.PlanStart,
			"plan_end":    objective.PlanEnd,
		}).Error
}

func (a *adapter) DeleteObjective(id int64) error {
	return a.db.
		Where("id = ?", id).
		Delete(&model.Objective{}).Error
}

func (a *adapter) DeleteObjectiveBySequence(uid types.Uid, topic string, sequence int64) error {
	return a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		Delete(&model.Objective{}).Error
}

func (a *adapter) GetKeyResultByID(id int64) (*model.KeyResult, error) {
	var keyResult model.KeyResult
	err := a.db.
		Where("id = ?", id).
		First(&keyResult).Error
	if err != nil {
		return nil, err
	}
	return &keyResult, nil
}

func (a *adapter) GetKeyResultBySequence(uid types.Uid, topic string, sequence int64) (*model.KeyResult, error) {
	var keyResult model.KeyResult
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		First(&keyResult).Error
	if err != nil {
		return nil, err
	}
	return &keyResult, nil
}

func (a *adapter) ListKeyResults(uid types.Uid, topic string) ([]*model.KeyResult, error) {
	var keyResult []*model.KeyResult
	err := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic).
		Order("id DESC").
		Find(&keyResult).Error
	if err != nil {
		return nil, err
	}
	return keyResult, nil
}

func (a *adapter) ListKeyResultsById(id []int64) ([]*model.KeyResult, error) {
	var keyResult []*model.KeyResult
	err := a.db.
		Where("id IN ?", id).
		Order("id DESC").
		Find(&keyResult).Error
	if err != nil {
		return nil, err
	}
	return keyResult, nil
}

func (a *adapter) ListKeyResultsByObjectiveId(objectiveId int64) ([]*model.KeyResult, error) {
	var keyResult []*model.KeyResult
	err := a.db.
		Where("objective_id = ?", objectiveId).
		Order("id DESC").
		Find(&keyResult).Error
	if err != nil {
		return nil, err
	}
	return keyResult, nil
}

func (a *adapter) CreateKeyResult(keyResult *model.KeyResult) (int64, error) {
	ctx := context.Background()
	l := locker.NewLocker()
	lock, err := l.Acquire(ctx, fmt.Sprintf("chatbot:key_result:create:%s", keyResult.UID), 10*time.Second)
	if err != nil {
		return 0, err
	}
	defer func(lock *redislock.Lock, ctx context.Context) {
		_ = lock.Release(ctx)
	}(lock, ctx)

	// sequence
	sequence := int32(0)
	var maxKeyResult model.KeyResult
	err = a.db.
		Where("`uid` = ? AND `topic` = ?", keyResult.UID, keyResult.Topic).
		Order("sequence DESC").
		Take(&maxKeyResult).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if maxKeyResult.Sequence > 0 {
		sequence = maxKeyResult.Sequence
	}
	sequence += 1

	keyResult.Sequence = sequence
	err = a.db.Create(&keyResult).Error
	if err != nil {
		return 0, err
	}

	// init value record
	if keyResult.CurrentValue > 0 {
		err = a.db.Create(&model.KeyResultValue{
			KeyResultID: keyResult.ID,
			Value:       keyResult.CurrentValue,
		}).Error
		if err != nil {
			return 0, err
		}
	}

	return keyResult.ID, nil
}

func (a *adapter) UpdateKeyResult(keyResult *model.KeyResult) error {
	return a.db.Model(&model.KeyResult{}).
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", keyResult.UID, keyResult.Topic, keyResult.Sequence).
		UpdateColumns(map[string]interface{}{
			"title":        keyResult.Title,
			"memo":         keyResult.Memo,
			"target_value": keyResult.TargetValue,
			"value_mode":   keyResult.ValueMode,
		}).Error
}

func (a *adapter) DeleteKeyResult(id int64) error {
	return a.db.
		Where("id = ?", id).
		Delete(&model.KeyResult{}).Error
}

func (a *adapter) DeleteKeyResultBySequence(uid types.Uid, topic string, sequence int64) error {
	return a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		Delete(&model.KeyResult{}).Error
}

func (a *adapter) AggregateObjectiveValue(id int64) error {
	result := model.KeyResult{}
	err := a.db.
		Model(&model.KeyResult{}).
		Where("objective_id = ?", id).
		Select("SUM(current_value) as current_value, SUM(target_value) as target_value").
		Take(&result).Error
	if err != nil {
		return err
	}
	return a.db.Model(&model.Objective{}).Where("id = ?", id).UpdateColumns(map[string]interface{}{
		"current_value": result.CurrentValue,
		"total_value":   result.TargetValue,
	}).Error
}

func (a *adapter) AggregateKeyResultValue(id int64) error {
	keyResult, err := a.GetKeyResultByID(id)
	if err != nil {
		return err
	}
	var value sql.NullInt64
	switch keyResult.ValueMode {
	case model.ValueSumMode:
		err = a.db.
			Model(&model.KeyResultValue{}).
			Where("key_result_id = ?", id).
			Select("SUM(`value`) as `value`").
			Pluck("value", &value).Error
	case model.ValueLastMode:
		err = a.db.
			Model(&model.KeyResultValue{}).
			Where("key_result_id = ?", id).
			Order("created_at DESC").
			Limit(1).
			Pluck("value", &value).Error
	case model.ValueAvgMode:
		err = a.db.
			Model(&model.KeyResultValue{}).
			Where("key_result_id = ?", id).
			Select("AVG(`value`) as `value`").
			Pluck("value", &value).Error
	case model.ValueMaxMode:
		err = a.db.
			Model(&model.KeyResultValue{}).
			Where("key_result_id = ?", id).
			Select("MAX(`value`) as `value`").
			Pluck("value", &value).Error
	}
	if err != nil {
		return err
	}

	return a.db.
		Model(&model.KeyResult{}).
		Where("id = ?", id).
		UpdateColumns(map[string]interface{}{
			"current_value": value.Int64,
		}).Error
}

func (a *adapter) CreateKeyResultValue(keyResultValue *model.KeyResultValue) (int64, error) {
	err := a.db.Create(&keyResultValue).Error
	if err != nil {
		return 0, err
	}
	return keyResultValue.ID, nil
}

func (a *adapter) DeleteKeyResultValue(id int64) error {
	q := dao.Q.KeyResultValue
	_, err := q.
		Where(q.ID.Eq(id)).
		Delete()
	return err
}

func (a *adapter) GetKeyResultValues(keyResultId int64) ([]*model.KeyResultValue, error) {
	var values []*model.KeyResultValue
	err := a.db.
		Where("key_result_id = ?", keyResultId).
		Order("id DESC").
		Find(&values).Error
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (a *adapter) GetKeyResultValue(id int64) (*model.KeyResultValue, error) {
	q := dao.Q.KeyResultValue
	return q.Where(q.ID.Eq(id)).First()
}

func (a *adapter) CreateTodo(todo *model.Todo) (int64, error) {
	ctx := context.Background()
	l := locker.NewLocker()
	lock, err := l.Acquire(ctx, fmt.Sprintf("chatbot:todo:create:%s", todo.UID), 10*time.Second)
	if err != nil {
		return 0, err
	}
	defer func(lock *redislock.Lock, ctx context.Context) {
		_ = lock.Release(ctx)
	}(lock, ctx)

	// sequence
	sequence := int32(0)
	var maxTodo model.Todo
	err = a.db.
		Where("`uid` = ? AND `topic` = ?", todo.UID, todo.Topic).
		Order("sequence DESC").
		Take(&maxTodo).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if maxTodo.Sequence > 0 {
		sequence = maxTodo.Sequence
	}
	sequence += 1

	todo.Sequence = sequence
	err = a.db.Create(&todo).Error
	if err != nil {
		return 0, nil
	}
	return todo.ID, nil
}

func (a *adapter) ListTodos(uid types.Uid, topic string) ([]*model.Todo, error) {
	var items []*model.Todo
	err := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic).
		Order("priority DESC").
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (a *adapter) ListRemindTodos(uid types.Uid, topic string) ([]*model.Todo, error) {
	var items []*model.Todo
	err := a.db.
		Where("`uid` = ? AND `topic` = ?", uid, topic).
		Where("complete <> ?", 1).
		Where("is_remind_at_time = ?", 1).
		Order("priority DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (a *adapter) GetTodo(id int64) (*model.Todo, error) {
	var find model.Todo
	err := a.db.
		Where("id = ?", id).
		First(&find).Error
	if err != nil {
		return nil, err
	}
	return &find, nil
}

func (a *adapter) GetTodoBySequence(uid types.Uid, topic string, sequence int64) (*model.Todo, error) {
	var find model.Todo
	err := a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		First(&find).Error
	if err != nil {
		return nil, err
	}
	return &find, nil
}

func (a *adapter) CompleteTodo(id int64) error {
	return a.db.Model(&model.Todo{}).
		Where("id = ?", id).
		Update("complete", true).Error
}

func (a *adapter) CompleteTodoBySequence(uid types.Uid, topic string, sequence int64) error {
	return a.db.Model(&model.Todo{}).
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		Update("complete", true).Error
}

func (a *adapter) UpdateTodo(todo *model.Todo) error {
	return a.db.Model(&model.Todo{}).
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", todo.UID, todo.Topic, todo.Sequence).
		UpdateColumns(map[string]interface{}{
			"content":  todo.Content,
			"category": todo.Category,
			"remark":   todo.Remark,
			"priority": todo.Priority,
		}).Error
}

func (a *adapter) DeleteTodo(id int64) error {
	return a.db.
		Where("id = ?", id).
		Delete(&model.Todo{}).Error
}

func (a *adapter) DeleteTodoBySequence(uid types.Uid, topic string, sequence int64) error {
	return a.db.
		Where("`uid` = ? AND `topic` = ? AND sequence = ?", uid, topic, sequence).
		Delete(&model.Todo{}).Error
}

func (a *adapter) CreateReview(review *model.Review) (int64, error) {
	q := dao.Q.Review
	err := q.Create(review)
	if err != nil {
		return 0, err
	}
	return review.ID, nil
}

func (a *adapter) UpdateReview(review *model.Review) error {
	q := dao.Q.Review
	_, err := q.Updates(review)
	return err
}

func (a *adapter) ListReviews(uid types.Uid, topic string) ([]*model.Review, error) {
	q := dao.Q.Review
	return q.
		Where(q.UID.Eq(string(uid)), q.Topic.Eq(topic)).
		Order(q.UpdatedAt.Desc()).
		Find()
}

func (a *adapter) GetReviewByID(id int64) (*model.Review, error) {
	q := dao.Q.Review
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) CreateReviewEvaluation(evaluation *model.ReviewEvaluation) (int64, error) {
	q := dao.Q.ReviewEvaluation
	err := q.Create(evaluation)
	if err != nil {
		return 0, err
	}
	return evaluation.ID, nil
}

func (a *adapter) UpdateReviewEvaluation(evaluation *model.ReviewEvaluation) error {
	q := dao.Q.ReviewEvaluation
	_, err := q.Updates(evaluation)
	return err
}

func (a *adapter) ListReviewEvaluations(uid types.Uid, topic string, reviewID int64) ([]*model.ReviewEvaluation, error) {
	q := dao.Q.ReviewEvaluation
	return q.
		Where(q.UID.Eq(string(uid)), q.Topic.Eq(topic), q.ReviewID.Eq(reviewID)).
		Order(q.UpdatedAt.Desc()).
		Find()
}

func (a *adapter) GetReviewEvaluationByID(id int64) (*model.ReviewEvaluation, error) {
	q := dao.Q.ReviewEvaluation
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) CreateCycle(cycle *model.Cycle) (int64, error) {
	q := dao.Q.Cycle
	err := q.Create(cycle)
	if err != nil {
		return 0, err
	}
	return cycle.ID, nil
}

func (a *adapter) UpdateCycle(cycle *model.Cycle) error {
	q := dao.Q.Cycle
	_, err := q.Updates(cycle)
	return err
}

func (a *adapter) ListCycles(uid types.Uid, topic string) ([]*model.Cycle, error) {
	q := dao.Q.Cycle
	return q.
		Where(q.UID.Eq(string(uid)), q.Topic.Eq(topic)).
		Order(q.UpdatedAt.Desc()).
		Find()
}

func (a *adapter) GetCycleByID(id int64) (*model.Cycle, error) {
	q := dao.Q.Cycle
	return q.Where(q.ID.Eq(id)).First()
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

func (a *adapter) CreateWorkflow(workflow *model.Workflow, script *model.WorkflowScript, dag *model.Dag, triggers []*model.WorkflowTrigger) (int64, error) {
	q := dao.Q
	err := q.Transaction(func(tx *dao.Query) error {
		err := tx.Workflow.Create(workflow)
		if err != nil {
			return err
		}

		if script != nil {
			script.WorkflowID = workflow.ID
			err = tx.WorkflowScript.
				Create(script)
			if err != nil {
				return err
			}

			if dag != nil {
				dag.WorkflowID = workflow.ID
				dag.ScriptID = script.ID
				dag.ScriptVersion = script.Version
				err = tx.Dag.
					Create(dag)
				if err != nil {
					return err
				}
			}
		}

		if len(triggers) > 0 {
			for _, trigger := range triggers {
				trigger.WorkflowID = workflow.ID
				err = tx.WorkflowTrigger.
					Create(trigger)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return workflow.ID, nil
}

func (a *adapter) GetWorkflow(id int64) (*model.Workflow, error) {
	q := dao.Q.Workflow
	return q.
		Preload(q.Dag, q.Triggers).
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) UpdateWorkflow(workflow *model.Workflow, script *model.WorkflowScript, dag *model.Dag, triggers []*model.WorkflowTrigger) error {
	q := dao.Q
	return q.Transaction(func(tx *dao.Query) error {
		_, err := tx.Workflow.
			Where(tx.Workflow.UID.Eq(workflow.UID), tx.Workflow.Topic.Eq(workflow.Topic), tx.Workflow.ID.Eq(workflow.ID)).
			UpdateColumns(workflow)
		if err != nil {
			return err
		}

		if script != nil {
			version := script.Version
			script.Version += 1
			_, err = tx.WorkflowScript.
				Where(tx.WorkflowScript.WorkflowID.Eq(workflow.ID), tx.WorkflowScript.Version.Eq(version)).
				UpdateColumns(script)
			if err != nil {
				return err
			}

			if dag != nil {
				dag.ScriptVersion = script.Version
				_, err = tx.Dag.
					Where(tx.Dag.WorkflowID.Eq(workflow.ID)).
					UpdateColumns(dag)
				if err != nil {
					return err
				}
			}
		}

		if len(triggers) > 0 {
			_, err = tx.WorkflowTrigger.
				Where(tx.WorkflowTrigger.WorkflowID.Eq(workflow.ID)).
				Delete()
			if err != nil {
				return err
			}
			for _, trigger := range triggers {
				trigger.WorkflowID = workflow.ID
				err = tx.WorkflowTrigger.Create(trigger)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (a *adapter) UpdateWorkflowState(id int64, state model.WorkflowState) error {
	q := dao.Q.Workflow
	_, err := q.
		Where(q.ID.Eq(id)).
		UpdateSimple(q.State.Value(state))
	return err
}

func (a *adapter) ListWorkflows(uid types.Uid, topic string) ([]*model.Workflow, error) {
	q := dao.Q.Workflow
	return q.
		Where(q.UID.Eq(string(uid))).
		Where(q.Topic.Eq(topic)).
		Preload(q.Triggers).
		Order(q.UpdatedAt.Desc()).
		Find()
}

func (a *adapter) IncreaseWorkflowCount(id int64, successful int32, failed int32, running int32, canceled int32) error {
	q := dao.Q.Workflow
	_, err := q.
		Where(q.ID.Eq(id)).
		UpdateSimple(
			q.SuccessfulCount.Add(successful),
			q.FailedCount.Add(failed),
			q.RunningCount.Add(running),
			q.CanceledCount.Add(canceled),
		)
	return err
}

func (a *adapter) IncreaseWorkflowTriggerCount(id int64, count int32) error {
	q := dao.Q.WorkflowTrigger
	_, err := q.
		Where(q.ID.Eq(id)).
		UpdateSimple(
			q.Count_.Add(count),
		)
	return err
}

func (a *adapter) DeleteWorkflow(id int64) error {
	q := dao.Q.Workflow
	_, err := q.
		Where(q.ID.Eq(id)).
		Delete()
	return err
}

func (a *adapter) CreateWorkflowScript(item *model.WorkflowScript) (int64, error) {
	q := dao.Q.WorkflowScript
	err := q.Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) GetWorkflowScript(id int64) (any, error) {
	q := dao.Q.WorkflowScript
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetWorkflowScriptByWorkflowId(workflowId int64) (any, error) {
	q := dao.Q.WorkflowScript
	return q.
		Where(q.WorkflowID.Eq(workflowId)).
		First()
}

func (a *adapter) CreateWorkflowTrigger(item *model.WorkflowTrigger) (int64, error) {
	q := dao.Q.WorkflowTrigger
	err := q.Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) UpdateWorkflowTrigger(item *model.WorkflowTrigger) error {
	q := dao.Q.WorkflowTrigger
	_, err := q.
		Where(q.ID.Eq(item.ID)).
		UpdateColumns(item)
	return err
}

func (a *adapter) DeleteWorkflowTrigger(id int64) error {
	q := dao.Q.WorkflowTrigger
	_, err := q.Where(q.ID.Eq(id)).Delete()
	return err
}

func (a *adapter) ListWorkflowTriggerByType(t model.TriggerType) ([]*model.WorkflowTrigger, error) {
	q := dao.Q.WorkflowTrigger
	return q.
		Where(q.Type.Eq(t)).
		Find()
}

func (a *adapter) UpdateDag(item *model.Dag) error {
	q := dao.Q.Dag
	_, err := q.
		Where(q.ID.Eq(item.ID)).
		UpdateColumns(item)
	return err
}

func (a *adapter) GetDag(id int64) (*model.Dag, error) {
	q := dao.Q.Dag
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) GetJob(id int64) (*model.Job, error) {
	q := dao.Q.Job
	return q.
		Where(q.ID.Eq(id)).
		First()
}

func (a *adapter) CreateJob(item *model.Job) (int64, error) {
	q := dao.Q.Job
	err := q.Create(item)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *adapter) DeleteJob(id int64) error {
	q := dao.Q.Job
	_, err := q.
		Where(q.ID.Eq(id)).
		Delete()
	return err
}

func (a *adapter) ListJobs(workflowID int64) ([]*model.Job, error) {
	q := dao.Q.Job
	return q.Where(q.WorkflowID.Eq(workflowID)).Find()
}

func (a *adapter) GetJobsByState(state model.JobState) ([]*model.Job, error) {
	q := dao.Q.Job
	return q.Where(q.State.Eq(state)).Find()
}

func (a *adapter) GetJobsByStates(states []model.JobState) ([]*model.Job, error) {
	q := dao.Q.Job
	build := q.Order(q.ID)
	for _, state := range states {
		build = build.
			Or(q.State.Eq(state))
	}
	return build.Find()
}

func (a *adapter) GetJobsByWorkflowId(workflowID int64) ([]*model.Job, error) {
	q := dao.Q.Job
	return q.
		Where(q.WorkflowID.Eq(workflowID)).
		Find()
}

func (a *adapter) UpdateJobState(id int64, state model.JobState) error {
	q := dao.Q.Job
	var err error
	switch state {
	case model.JobStart:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state), q.StartedAt.Value(time.Now()))
	case model.JobSucceeded, model.JobFailed, model.JobCanceled:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state), q.EndedAt.Value(time.Now()))
	default:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state))
	}
	return err
}

func (a *adapter) UpdateStep(id int64, item *model.Step) error {
	q := dao.Q.Step
	_, err := q.
		Where(q.ID.Eq(id)).
		Updates(item)
	return err
}

func (a *adapter) UpdateStepState(id int64, state model.StepState) error {
	q := dao.Q.Step
	var err error
	switch state {
	case model.StepStart:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state), q.StartedAt.Value(time.Now()))
	case model.StepSucceeded, model.StepFailed, model.StepCanceled, model.StepSkipped:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state), q.EndedAt.Value(time.Now()))
	default:
		_, err = q.
			Where(q.ID.Eq(id)).
			UpdateSimple(q.State.Value(state))
	}
	return err
}

func (a *adapter) UpdateStepInput(id int64, input types.KV) error {
	q := dao.Q.Step
	_, err := q.
		Where(q.ID.Eq(id)).
		Update(q.Input, input)
	return err
}

func (a *adapter) UpdateStepOutput(id int64, output types.KV) error {
	q := dao.Q.Step
	_, err := q.
		Where(q.ID.Eq(id)).
		Update(q.Output, output)
	return err
}

func (a *adapter) CreateStep(step *model.Step) (int64, error) {
	q := dao.Q.Step
	err := q.Create(step)
	if err != nil {
		return 0, err
	}
	return step.ID, nil
}

func (a *adapter) CreateSteps(steps []*model.Step) error {
	q := dao.Q
	return q.Transaction(func(tx *dao.Query) error {
		for _, step := range steps {
			err := tx.Step.Create(step)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (a *adapter) GetStepsByState(state model.StepState) ([]*model.Step, error) {
	q := dao.Q.Step
	return q.
		Where(q.State.Eq(state)).
		Find()
}

func (a *adapter) GetStepsByDepend(jobId int64, depend []string) ([]*model.Step, error) {
	q := dao.Q.Step
	return q.
		Where(q.JobID.Eq(jobId), q.Columns(q.NodeID).In(field.Values(depend))).
		Find()
}

func (a *adapter) GetStepsByJobId(jobId int64) ([]*model.Step, error) {
	q := dao.Q.Step
	return q.
		Where(q.JobID.Eq(jobId)).
		Find()
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

func Init() {
	store.RegisterAdapter(&adapter{})
}

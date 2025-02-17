// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package dao

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"

	"gorm.io/plugin/dbresolver"

	"github.com/flowline-io/flowbot/internal/store/model"
)

func newAgent(db *gorm.DB, opts ...gen.DOOption) agent {
	_agent := agent{}

	_agent.agentDo.UseDB(db, opts...)
	_agent.agentDo.UseModel(&model.Agent{})

	tableName := _agent.agentDo.TableName()
	_agent.ALL = field.NewAsterisk(tableName)
	_agent.ID = field.NewInt64(tableName, "id")
	_agent.UID = field.NewString(tableName, "uid")
	_agent.Topic = field.NewString(tableName, "topic")
	_agent.Hostid = field.NewString(tableName, "hostid")
	_agent.Hostname = field.NewString(tableName, "hostname")
	_agent.OnlineDuration = field.NewInt32(tableName, "online_duration")
	_agent.LastOnlineAt = field.NewTime(tableName, "last_online_at")
	_agent.CreatedAt = field.NewTime(tableName, "created_at")
	_agent.UpdatedAt = field.NewTime(tableName, "updated_at")

	_agent.fillFieldMap()

	return _agent
}

type agent struct {
	agentDo

	ALL            field.Asterisk
	ID             field.Int64
	UID            field.String
	Topic          field.String
	Hostid         field.String
	Hostname       field.String
	OnlineDuration field.Int32
	LastOnlineAt   field.Time
	CreatedAt      field.Time
	UpdatedAt      field.Time

	fieldMap map[string]field.Expr
}

func (a agent) Table(newTableName string) *agent {
	a.agentDo.UseTable(newTableName)
	return a.updateTableName(newTableName)
}

func (a agent) As(alias string) *agent {
	a.agentDo.DO = *(a.agentDo.As(alias).(*gen.DO))
	return a.updateTableName(alias)
}

func (a *agent) updateTableName(table string) *agent {
	a.ALL = field.NewAsterisk(table)
	a.ID = field.NewInt64(table, "id")
	a.UID = field.NewString(table, "uid")
	a.Topic = field.NewString(table, "topic")
	a.Hostid = field.NewString(table, "hostid")
	a.Hostname = field.NewString(table, "hostname")
	a.OnlineDuration = field.NewInt32(table, "online_duration")
	a.LastOnlineAt = field.NewTime(table, "last_online_at")
	a.CreatedAt = field.NewTime(table, "created_at")
	a.UpdatedAt = field.NewTime(table, "updated_at")

	a.fillFieldMap()

	return a
}

func (a *agent) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := a.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (a *agent) fillFieldMap() {
	a.fieldMap = make(map[string]field.Expr, 9)
	a.fieldMap["id"] = a.ID
	a.fieldMap["uid"] = a.UID
	a.fieldMap["topic"] = a.Topic
	a.fieldMap["hostid"] = a.Hostid
	a.fieldMap["hostname"] = a.Hostname
	a.fieldMap["online_duration"] = a.OnlineDuration
	a.fieldMap["last_online_at"] = a.LastOnlineAt
	a.fieldMap["created_at"] = a.CreatedAt
	a.fieldMap["updated_at"] = a.UpdatedAt
}

func (a agent) clone(db *gorm.DB) agent {
	a.agentDo.ReplaceConnPool(db.Statement.ConnPool)
	return a
}

func (a agent) replaceDB(db *gorm.DB) agent {
	a.agentDo.ReplaceDB(db)
	return a
}

type agentDo struct{ gen.DO }

func (a agentDo) Debug() *agentDo {
	return a.withDO(a.DO.Debug())
}

func (a agentDo) WithContext(ctx context.Context) *agentDo {
	return a.withDO(a.DO.WithContext(ctx))
}

func (a agentDo) ReadDB() *agentDo {
	return a.Clauses(dbresolver.Read)
}

func (a agentDo) WriteDB() *agentDo {
	return a.Clauses(dbresolver.Write)
}

func (a agentDo) Session(config *gorm.Session) *agentDo {
	return a.withDO(a.DO.Session(config))
}

func (a agentDo) Clauses(conds ...clause.Expression) *agentDo {
	return a.withDO(a.DO.Clauses(conds...))
}

func (a agentDo) Returning(value interface{}, columns ...string) *agentDo {
	return a.withDO(a.DO.Returning(value, columns...))
}

func (a agentDo) Not(conds ...gen.Condition) *agentDo {
	return a.withDO(a.DO.Not(conds...))
}

func (a agentDo) Or(conds ...gen.Condition) *agentDo {
	return a.withDO(a.DO.Or(conds...))
}

func (a agentDo) Select(conds ...field.Expr) *agentDo {
	return a.withDO(a.DO.Select(conds...))
}

func (a agentDo) Where(conds ...gen.Condition) *agentDo {
	return a.withDO(a.DO.Where(conds...))
}

func (a agentDo) Order(conds ...field.Expr) *agentDo {
	return a.withDO(a.DO.Order(conds...))
}

func (a agentDo) Distinct(cols ...field.Expr) *agentDo {
	return a.withDO(a.DO.Distinct(cols...))
}

func (a agentDo) Omit(cols ...field.Expr) *agentDo {
	return a.withDO(a.DO.Omit(cols...))
}

func (a agentDo) Join(table schema.Tabler, on ...field.Expr) *agentDo {
	return a.withDO(a.DO.Join(table, on...))
}

func (a agentDo) LeftJoin(table schema.Tabler, on ...field.Expr) *agentDo {
	return a.withDO(a.DO.LeftJoin(table, on...))
}

func (a agentDo) RightJoin(table schema.Tabler, on ...field.Expr) *agentDo {
	return a.withDO(a.DO.RightJoin(table, on...))
}

func (a agentDo) Group(cols ...field.Expr) *agentDo {
	return a.withDO(a.DO.Group(cols...))
}

func (a agentDo) Having(conds ...gen.Condition) *agentDo {
	return a.withDO(a.DO.Having(conds...))
}

func (a agentDo) Limit(limit int) *agentDo {
	return a.withDO(a.DO.Limit(limit))
}

func (a agentDo) Offset(offset int) *agentDo {
	return a.withDO(a.DO.Offset(offset))
}

func (a agentDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *agentDo {
	return a.withDO(a.DO.Scopes(funcs...))
}

func (a agentDo) Unscoped() *agentDo {
	return a.withDO(a.DO.Unscoped())
}

func (a agentDo) Create(values ...*model.Agent) error {
	if len(values) == 0 {
		return nil
	}
	return a.DO.Create(values)
}

func (a agentDo) CreateInBatches(values []*model.Agent, batchSize int) error {
	return a.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (a agentDo) Save(values ...*model.Agent) error {
	if len(values) == 0 {
		return nil
	}
	return a.DO.Save(values)
}

func (a agentDo) First() (*model.Agent, error) {
	if result, err := a.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Agent), nil
	}
}

func (a agentDo) Take() (*model.Agent, error) {
	if result, err := a.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Agent), nil
	}
}

func (a agentDo) Last() (*model.Agent, error) {
	if result, err := a.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Agent), nil
	}
}

func (a agentDo) Find() ([]*model.Agent, error) {
	result, err := a.DO.Find()
	return result.([]*model.Agent), err
}

func (a agentDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Agent, err error) {
	buf := make([]*model.Agent, 0, batchSize)
	err = a.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (a agentDo) FindInBatches(result *[]*model.Agent, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return a.DO.FindInBatches(result, batchSize, fc)
}

func (a agentDo) Attrs(attrs ...field.AssignExpr) *agentDo {
	return a.withDO(a.DO.Attrs(attrs...))
}

func (a agentDo) Assign(attrs ...field.AssignExpr) *agentDo {
	return a.withDO(a.DO.Assign(attrs...))
}

func (a agentDo) Joins(fields ...field.RelationField) *agentDo {
	for _, _f := range fields {
		a = *a.withDO(a.DO.Joins(_f))
	}
	return &a
}

func (a agentDo) Preload(fields ...field.RelationField) *agentDo {
	for _, _f := range fields {
		a = *a.withDO(a.DO.Preload(_f))
	}
	return &a
}

func (a agentDo) FirstOrInit() (*model.Agent, error) {
	if result, err := a.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Agent), nil
	}
}

func (a agentDo) FirstOrCreate() (*model.Agent, error) {
	if result, err := a.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Agent), nil
	}
}

func (a agentDo) FindByPage(offset int, limit int) (result []*model.Agent, count int64, err error) {
	result, err = a.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = a.Offset(-1).Limit(-1).Count()
	return
}

func (a agentDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = a.Count()
	if err != nil {
		return
	}

	err = a.Offset(offset).Limit(limit).Scan(result)
	return
}

func (a agentDo) Scan(result interface{}) (err error) {
	return a.DO.Scan(result)
}

func (a agentDo) Delete(models ...*model.Agent) (result gen.ResultInfo, err error) {
	return a.DO.Delete(models)
}

func (a *agentDo) withDO(do gen.Dao) *agentDo {
	a.DO = *do.(*gen.DO)
	return a
}

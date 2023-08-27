// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package dao

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"

	"gorm.io/plugin/dbresolver"

	"github.com/flowline-io/flowbot/internal/store/model"
)

func newCycle(db *gorm.DB, opts ...gen.DOOption) cycle {
	_cycle := cycle{}

	_cycle.cycleDo.UseDB(db, opts...)
	_cycle.cycleDo.UseModel(&model.Cycle{})

	tableName := _cycle.cycleDo.TableName()
	_cycle.ALL = field.NewAsterisk(tableName)
	_cycle.ID = field.NewInt32(tableName, "id")
	_cycle.UID = field.NewString(tableName, "uid")
	_cycle.Topic = field.NewString(tableName, "topic")
	_cycle.Objectives = field.NewString(tableName, "objectives")
	_cycle.StartDate = field.NewTime(tableName, "start_date")
	_cycle.EndDate = field.NewTime(tableName, "end_date")
	_cycle.State = field.NewField(tableName, "state")
	_cycle.CreatedAt = field.NewTime(tableName, "created_at")
	_cycle.UpdatedAt = field.NewTime(tableName, "updated_at")

	_cycle.fillFieldMap()

	return _cycle
}

type cycle struct {
	cycleDo

	ALL        field.Asterisk
	ID         field.Int32
	UID        field.String
	Topic      field.String
	Objectives field.String
	StartDate  field.Time
	EndDate    field.Time
	State      field.Field
	CreatedAt  field.Time
	UpdatedAt  field.Time

	fieldMap map[string]field.Expr
}

func (c cycle) Table(newTableName string) *cycle {
	c.cycleDo.UseTable(newTableName)
	return c.updateTableName(newTableName)
}

func (c cycle) As(alias string) *cycle {
	c.cycleDo.DO = *(c.cycleDo.As(alias).(*gen.DO))
	return c.updateTableName(alias)
}

func (c *cycle) updateTableName(table string) *cycle {
	c.ALL = field.NewAsterisk(table)
	c.ID = field.NewInt32(table, "id")
	c.UID = field.NewString(table, "uid")
	c.Topic = field.NewString(table, "topic")
	c.Objectives = field.NewString(table, "objectives")
	c.StartDate = field.NewTime(table, "start_date")
	c.EndDate = field.NewTime(table, "end_date")
	c.State = field.NewField(table, "state")
	c.CreatedAt = field.NewTime(table, "created_at")
	c.UpdatedAt = field.NewTime(table, "updated_at")

	c.fillFieldMap()

	return c
}

func (c *cycle) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := c.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (c *cycle) fillFieldMap() {
	c.fieldMap = make(map[string]field.Expr, 9)
	c.fieldMap["id"] = c.ID
	c.fieldMap["uid"] = c.UID
	c.fieldMap["topic"] = c.Topic
	c.fieldMap["objectives"] = c.Objectives
	c.fieldMap["start_date"] = c.StartDate
	c.fieldMap["end_date"] = c.EndDate
	c.fieldMap["state"] = c.State
	c.fieldMap["created_at"] = c.CreatedAt
	c.fieldMap["updated_at"] = c.UpdatedAt
}

func (c cycle) clone(db *gorm.DB) cycle {
	c.cycleDo.ReplaceConnPool(db.Statement.ConnPool)
	return c
}

func (c cycle) replaceDB(db *gorm.DB) cycle {
	c.cycleDo.ReplaceDB(db)
	return c
}

type cycleDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (c cycleDo) GetByID(id int) (result model.Cycle, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM chatbot_cycles WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = c.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (c cycleDo) TakeByUidAndTopic(uid string, topic string) (result model.Cycle, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = c.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (c cycleDo) FindByUidAndTopic(uid string, topic string) (result []*model.Cycle, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = c.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (c cycleDo) Debug() *cycleDo {
	return c.withDO(c.DO.Debug())
}

func (c cycleDo) WithContext(ctx context.Context) *cycleDo {
	return c.withDO(c.DO.WithContext(ctx))
}

func (c cycleDo) ReadDB() *cycleDo {
	return c.Clauses(dbresolver.Read)
}

func (c cycleDo) WriteDB() *cycleDo {
	return c.Clauses(dbresolver.Write)
}

func (c cycleDo) Session(config *gorm.Session) *cycleDo {
	return c.withDO(c.DO.Session(config))
}

func (c cycleDo) Clauses(conds ...clause.Expression) *cycleDo {
	return c.withDO(c.DO.Clauses(conds...))
}

func (c cycleDo) Returning(value interface{}, columns ...string) *cycleDo {
	return c.withDO(c.DO.Returning(value, columns...))
}

func (c cycleDo) Not(conds ...gen.Condition) *cycleDo {
	return c.withDO(c.DO.Not(conds...))
}

func (c cycleDo) Or(conds ...gen.Condition) *cycleDo {
	return c.withDO(c.DO.Or(conds...))
}

func (c cycleDo) Select(conds ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Select(conds...))
}

func (c cycleDo) Where(conds ...gen.Condition) *cycleDo {
	return c.withDO(c.DO.Where(conds...))
}

func (c cycleDo) Order(conds ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Order(conds...))
}

func (c cycleDo) Distinct(cols ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Distinct(cols...))
}

func (c cycleDo) Omit(cols ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Omit(cols...))
}

func (c cycleDo) Join(table schema.Tabler, on ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Join(table, on...))
}

func (c cycleDo) LeftJoin(table schema.Tabler, on ...field.Expr) *cycleDo {
	return c.withDO(c.DO.LeftJoin(table, on...))
}

func (c cycleDo) RightJoin(table schema.Tabler, on ...field.Expr) *cycleDo {
	return c.withDO(c.DO.RightJoin(table, on...))
}

func (c cycleDo) Group(cols ...field.Expr) *cycleDo {
	return c.withDO(c.DO.Group(cols...))
}

func (c cycleDo) Having(conds ...gen.Condition) *cycleDo {
	return c.withDO(c.DO.Having(conds...))
}

func (c cycleDo) Limit(limit int) *cycleDo {
	return c.withDO(c.DO.Limit(limit))
}

func (c cycleDo) Offset(offset int) *cycleDo {
	return c.withDO(c.DO.Offset(offset))
}

func (c cycleDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *cycleDo {
	return c.withDO(c.DO.Scopes(funcs...))
}

func (c cycleDo) Unscoped() *cycleDo {
	return c.withDO(c.DO.Unscoped())
}

func (c cycleDo) Create(values ...*model.Cycle) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Create(values)
}

func (c cycleDo) CreateInBatches(values []*model.Cycle, batchSize int) error {
	return c.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (c cycleDo) Save(values ...*model.Cycle) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Save(values)
}

func (c cycleDo) First() (*model.Cycle, error) {
	if result, err := c.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Cycle), nil
	}
}

func (c cycleDo) Take() (*model.Cycle, error) {
	if result, err := c.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Cycle), nil
	}
}

func (c cycleDo) Last() (*model.Cycle, error) {
	if result, err := c.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Cycle), nil
	}
}

func (c cycleDo) Find() ([]*model.Cycle, error) {
	result, err := c.DO.Find()
	return result.([]*model.Cycle), err
}

func (c cycleDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Cycle, err error) {
	buf := make([]*model.Cycle, 0, batchSize)
	err = c.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (c cycleDo) FindInBatches(result *[]*model.Cycle, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return c.DO.FindInBatches(result, batchSize, fc)
}

func (c cycleDo) Attrs(attrs ...field.AssignExpr) *cycleDo {
	return c.withDO(c.DO.Attrs(attrs...))
}

func (c cycleDo) Assign(attrs ...field.AssignExpr) *cycleDo {
	return c.withDO(c.DO.Assign(attrs...))
}

func (c cycleDo) Joins(fields ...field.RelationField) *cycleDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Joins(_f))
	}
	return &c
}

func (c cycleDo) Preload(fields ...field.RelationField) *cycleDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Preload(_f))
	}
	return &c
}

func (c cycleDo) FirstOrInit() (*model.Cycle, error) {
	if result, err := c.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Cycle), nil
	}
}

func (c cycleDo) FirstOrCreate() (*model.Cycle, error) {
	if result, err := c.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Cycle), nil
	}
}

func (c cycleDo) FindByPage(offset int, limit int) (result []*model.Cycle, count int64, err error) {
	result, err = c.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = c.Offset(-1).Limit(-1).Count()
	return
}

func (c cycleDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = c.Count()
	if err != nil {
		return
	}

	err = c.Offset(offset).Limit(limit).Scan(result)
	return
}

func (c cycleDo) Scan(result interface{}) (err error) {
	return c.DO.Scan(result)
}

func (c cycleDo) Delete(models ...*model.Cycle) (result gen.ResultInfo, err error) {
	return c.DO.Delete(models)
}

func (c *cycleDo) withDO(do gen.Dao) *cycleDo {
	c.DO = *do.(*gen.DO)
	return c
}

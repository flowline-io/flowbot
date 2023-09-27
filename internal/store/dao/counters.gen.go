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

func newCounter(db *gorm.DB, opts ...gen.DOOption) counter {
	_counter := counter{}

	_counter.counterDo.UseDB(db, opts...)
	_counter.counterDo.UseModel(&model.Counter{})

	tableName := _counter.counterDo.TableName()
	_counter.ALL = field.NewAsterisk(tableName)
	_counter.ID = field.NewInt64(tableName, "id")
	_counter.UID = field.NewString(tableName, "uid")
	_counter.Topic = field.NewString(tableName, "topic")
	_counter.Flag = field.NewString(tableName, "flag")
	_counter.Digit = field.NewInt64(tableName, "digit")
	_counter.Status = field.NewInt32(tableName, "status")
	_counter.CreatedAt = field.NewTime(tableName, "created_at")
	_counter.UpdatedAt = field.NewTime(tableName, "updated_at")

	_counter.fillFieldMap()

	return _counter
}

type counter struct {
	counterDo

	ALL       field.Asterisk
	ID        field.Int64
	UID       field.String
	Topic     field.String
	Flag      field.String
	Digit     field.Int64
	Status    field.Int32
	CreatedAt field.Time
	UpdatedAt field.Time

	fieldMap map[string]field.Expr
}

func (c counter) Table(newTableName string) *counter {
	c.counterDo.UseTable(newTableName)
	return c.updateTableName(newTableName)
}

func (c counter) As(alias string) *counter {
	c.counterDo.DO = *(c.counterDo.As(alias).(*gen.DO))
	return c.updateTableName(alias)
}

func (c *counter) updateTableName(table string) *counter {
	c.ALL = field.NewAsterisk(table)
	c.ID = field.NewInt64(table, "id")
	c.UID = field.NewString(table, "uid")
	c.Topic = field.NewString(table, "topic")
	c.Flag = field.NewString(table, "flag")
	c.Digit = field.NewInt64(table, "digit")
	c.Status = field.NewInt32(table, "status")
	c.CreatedAt = field.NewTime(table, "created_at")
	c.UpdatedAt = field.NewTime(table, "updated_at")

	c.fillFieldMap()

	return c
}

func (c *counter) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := c.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (c *counter) fillFieldMap() {
	c.fieldMap = make(map[string]field.Expr, 8)
	c.fieldMap["id"] = c.ID
	c.fieldMap["uid"] = c.UID
	c.fieldMap["topic"] = c.Topic
	c.fieldMap["flag"] = c.Flag
	c.fieldMap["digit"] = c.Digit
	c.fieldMap["status"] = c.Status
	c.fieldMap["created_at"] = c.CreatedAt
	c.fieldMap["updated_at"] = c.UpdatedAt
}

func (c counter) clone(db *gorm.DB) counter {
	c.counterDo.ReplaceConnPool(db.Statement.ConnPool)
	return c
}

func (c counter) replaceDB(db *gorm.DB) counter {
	c.counterDo.ReplaceDB(db)
	return c
}

type counterDo struct{ gen.DO }

// GetByID
//
// SELECT  * FROM @@table WHERE id=@id
func (c counterDo) GetByID(id int) (result model.Counter, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM counters WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = c.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid  AND topic=@topic")
func (c counterDo) TakeByUidAndTopic(uid string, topic string) (result model.Counter, err error) {
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
// where("uid=@uid  AND topic=@topic")
func (c counterDo) FindByUidAndTopic(uid string, topic string) (result []*model.Counter, err error) {
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

func (c counterDo) Debug() *counterDo {
	return c.withDO(c.DO.Debug())
}

func (c counterDo) WithContext(ctx context.Context) *counterDo {
	return c.withDO(c.DO.WithContext(ctx))
}

func (c counterDo) ReadDB() *counterDo {
	return c.Clauses(dbresolver.Read)
}

func (c counterDo) WriteDB() *counterDo {
	return c.Clauses(dbresolver.Write)
}

func (c counterDo) Session(config *gorm.Session) *counterDo {
	return c.withDO(c.DO.Session(config))
}

func (c counterDo) Clauses(conds ...clause.Expression) *counterDo {
	return c.withDO(c.DO.Clauses(conds...))
}

func (c counterDo) Returning(value interface{}, columns ...string) *counterDo {
	return c.withDO(c.DO.Returning(value, columns...))
}

func (c counterDo) Not(conds ...gen.Condition) *counterDo {
	return c.withDO(c.DO.Not(conds...))
}

func (c counterDo) Or(conds ...gen.Condition) *counterDo {
	return c.withDO(c.DO.Or(conds...))
}

func (c counterDo) Select(conds ...field.Expr) *counterDo {
	return c.withDO(c.DO.Select(conds...))
}

func (c counterDo) Where(conds ...gen.Condition) *counterDo {
	return c.withDO(c.DO.Where(conds...))
}

func (c counterDo) Order(conds ...field.Expr) *counterDo {
	return c.withDO(c.DO.Order(conds...))
}

func (c counterDo) Distinct(cols ...field.Expr) *counterDo {
	return c.withDO(c.DO.Distinct(cols...))
}

func (c counterDo) Omit(cols ...field.Expr) *counterDo {
	return c.withDO(c.DO.Omit(cols...))
}

func (c counterDo) Join(table schema.Tabler, on ...field.Expr) *counterDo {
	return c.withDO(c.DO.Join(table, on...))
}

func (c counterDo) LeftJoin(table schema.Tabler, on ...field.Expr) *counterDo {
	return c.withDO(c.DO.LeftJoin(table, on...))
}

func (c counterDo) RightJoin(table schema.Tabler, on ...field.Expr) *counterDo {
	return c.withDO(c.DO.RightJoin(table, on...))
}

func (c counterDo) Group(cols ...field.Expr) *counterDo {
	return c.withDO(c.DO.Group(cols...))
}

func (c counterDo) Having(conds ...gen.Condition) *counterDo {
	return c.withDO(c.DO.Having(conds...))
}

func (c counterDo) Limit(limit int) *counterDo {
	return c.withDO(c.DO.Limit(limit))
}

func (c counterDo) Offset(offset int) *counterDo {
	return c.withDO(c.DO.Offset(offset))
}

func (c counterDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *counterDo {
	return c.withDO(c.DO.Scopes(funcs...))
}

func (c counterDo) Unscoped() *counterDo {
	return c.withDO(c.DO.Unscoped())
}

func (c counterDo) Create(values ...*model.Counter) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Create(values)
}

func (c counterDo) CreateInBatches(values []*model.Counter, batchSize int) error {
	return c.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (c counterDo) Save(values ...*model.Counter) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Save(values)
}

func (c counterDo) First() (*model.Counter, error) {
	if result, err := c.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Counter), nil
	}
}

func (c counterDo) Take() (*model.Counter, error) {
	if result, err := c.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Counter), nil
	}
}

func (c counterDo) Last() (*model.Counter, error) {
	if result, err := c.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Counter), nil
	}
}

func (c counterDo) Find() ([]*model.Counter, error) {
	result, err := c.DO.Find()
	return result.([]*model.Counter), err
}

func (c counterDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Counter, err error) {
	buf := make([]*model.Counter, 0, batchSize)
	err = c.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (c counterDo) FindInBatches(result *[]*model.Counter, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return c.DO.FindInBatches(result, batchSize, fc)
}

func (c counterDo) Attrs(attrs ...field.AssignExpr) *counterDo {
	return c.withDO(c.DO.Attrs(attrs...))
}

func (c counterDo) Assign(attrs ...field.AssignExpr) *counterDo {
	return c.withDO(c.DO.Assign(attrs...))
}

func (c counterDo) Joins(fields ...field.RelationField) *counterDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Joins(_f))
	}
	return &c
}

func (c counterDo) Preload(fields ...field.RelationField) *counterDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Preload(_f))
	}
	return &c
}

func (c counterDo) FirstOrInit() (*model.Counter, error) {
	if result, err := c.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Counter), nil
	}
}

func (c counterDo) FirstOrCreate() (*model.Counter, error) {
	if result, err := c.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Counter), nil
	}
}

func (c counterDo) FindByPage(offset int, limit int) (result []*model.Counter, count int64, err error) {
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

func (c counterDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = c.Count()
	if err != nil {
		return
	}

	err = c.Offset(offset).Limit(limit).Scan(result)
	return
}

func (c counterDo) Scan(result interface{}) (err error) {
	return c.DO.Scan(result)
}

func (c counterDo) Delete(models ...*model.Counter) (result gen.ResultInfo, err error) {
	return c.DO.Delete(models)
}

func (c *counterDo) withDO(do gen.Dao) *counterDo {
	c.DO = *do.(*gen.DO)
	return c
}

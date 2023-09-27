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

func newStep(db *gorm.DB, opts ...gen.DOOption) step {
	_step := step{}

	_step.stepDo.UseDB(db, opts...)
	_step.stepDo.UseModel(&model.Step{})

	tableName := _step.stepDo.TableName()
	_step.ALL = field.NewAsterisk(tableName)
	_step.ID = field.NewInt64(tableName, "id")
	_step.UID = field.NewString(tableName, "uid")
	_step.Topic = field.NewString(tableName, "topic")
	_step.JobID = field.NewInt64(tableName, "job_id")
	_step.Action = field.NewField(tableName, "action")
	_step.Name = field.NewString(tableName, "name")
	_step.Describe = field.NewString(tableName, "describe")
	_step.NodeID = field.NewString(tableName, "node_id")
	_step.Depend = field.NewField(tableName, "depend")
	_step.Input = field.NewField(tableName, "input")
	_step.Output = field.NewField(tableName, "output")
	_step.Error = field.NewString(tableName, "error")
	_step.State = field.NewField(tableName, "state")
	_step.StartedAt = field.NewTime(tableName, "started_at")
	_step.FinishedAt = field.NewTime(tableName, "finished_at")
	_step.CreatedAt = field.NewTime(tableName, "created_at")
	_step.UpdatedAt = field.NewTime(tableName, "updated_at")

	_step.fillFieldMap()

	return _step
}

type step struct {
	stepDo

	ALL        field.Asterisk
	ID         field.Int64
	UID        field.String
	Topic      field.String
	JobID      field.Int64
	Action     field.Field
	Name       field.String
	Describe   field.String
	NodeID     field.String
	Depend     field.Field
	Input      field.Field
	Output     field.Field
	Error      field.String
	State      field.Field
	StartedAt  field.Time
	FinishedAt field.Time
	CreatedAt  field.Time
	UpdatedAt  field.Time

	fieldMap map[string]field.Expr
}

func (s step) Table(newTableName string) *step {
	s.stepDo.UseTable(newTableName)
	return s.updateTableName(newTableName)
}

func (s step) As(alias string) *step {
	s.stepDo.DO = *(s.stepDo.As(alias).(*gen.DO))
	return s.updateTableName(alias)
}

func (s *step) updateTableName(table string) *step {
	s.ALL = field.NewAsterisk(table)
	s.ID = field.NewInt64(table, "id")
	s.UID = field.NewString(table, "uid")
	s.Topic = field.NewString(table, "topic")
	s.JobID = field.NewInt64(table, "job_id")
	s.Action = field.NewField(table, "action")
	s.Name = field.NewString(table, "name")
	s.Describe = field.NewString(table, "describe")
	s.NodeID = field.NewString(table, "node_id")
	s.Depend = field.NewField(table, "depend")
	s.Input = field.NewField(table, "input")
	s.Output = field.NewField(table, "output")
	s.Error = field.NewString(table, "error")
	s.State = field.NewField(table, "state")
	s.StartedAt = field.NewTime(table, "started_at")
	s.FinishedAt = field.NewTime(table, "finished_at")
	s.CreatedAt = field.NewTime(table, "created_at")
	s.UpdatedAt = field.NewTime(table, "updated_at")

	s.fillFieldMap()

	return s
}

func (s *step) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := s.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (s *step) fillFieldMap() {
	s.fieldMap = make(map[string]field.Expr, 17)
	s.fieldMap["id"] = s.ID
	s.fieldMap["uid"] = s.UID
	s.fieldMap["topic"] = s.Topic
	s.fieldMap["job_id"] = s.JobID
	s.fieldMap["action"] = s.Action
	s.fieldMap["name"] = s.Name
	s.fieldMap["describe"] = s.Describe
	s.fieldMap["node_id"] = s.NodeID
	s.fieldMap["depend"] = s.Depend
	s.fieldMap["input"] = s.Input
	s.fieldMap["output"] = s.Output
	s.fieldMap["error"] = s.Error
	s.fieldMap["state"] = s.State
	s.fieldMap["started_at"] = s.StartedAt
	s.fieldMap["finished_at"] = s.FinishedAt
	s.fieldMap["created_at"] = s.CreatedAt
	s.fieldMap["updated_at"] = s.UpdatedAt
}

func (s step) clone(db *gorm.DB) step {
	s.stepDo.ReplaceConnPool(db.Statement.ConnPool)
	return s
}

func (s step) replaceDB(db *gorm.DB) step {
	s.stepDo.ReplaceDB(db)
	return s
}

type stepDo struct{ gen.DO }

// GetByID
//
// SELECT  * FROM @@table WHERE id=@id
func (s stepDo) GetByID(id int) (result model.Step, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM steps WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = s.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid  AND topic=@topic")
func (s stepDo) TakeByUidAndTopic(uid string, topic string) (result model.Step, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = s.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid  AND topic=@topic")
func (s stepDo) FindByUidAndTopic(uid string, topic string) (result []*model.Step, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = s.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (s stepDo) Debug() *stepDo {
	return s.withDO(s.DO.Debug())
}

func (s stepDo) WithContext(ctx context.Context) *stepDo {
	return s.withDO(s.DO.WithContext(ctx))
}

func (s stepDo) ReadDB() *stepDo {
	return s.Clauses(dbresolver.Read)
}

func (s stepDo) WriteDB() *stepDo {
	return s.Clauses(dbresolver.Write)
}

func (s stepDo) Session(config *gorm.Session) *stepDo {
	return s.withDO(s.DO.Session(config))
}

func (s stepDo) Clauses(conds ...clause.Expression) *stepDo {
	return s.withDO(s.DO.Clauses(conds...))
}

func (s stepDo) Returning(value interface{}, columns ...string) *stepDo {
	return s.withDO(s.DO.Returning(value, columns...))
}

func (s stepDo) Not(conds ...gen.Condition) *stepDo {
	return s.withDO(s.DO.Not(conds...))
}

func (s stepDo) Or(conds ...gen.Condition) *stepDo {
	return s.withDO(s.DO.Or(conds...))
}

func (s stepDo) Select(conds ...field.Expr) *stepDo {
	return s.withDO(s.DO.Select(conds...))
}

func (s stepDo) Where(conds ...gen.Condition) *stepDo {
	return s.withDO(s.DO.Where(conds...))
}

func (s stepDo) Order(conds ...field.Expr) *stepDo {
	return s.withDO(s.DO.Order(conds...))
}

func (s stepDo) Distinct(cols ...field.Expr) *stepDo {
	return s.withDO(s.DO.Distinct(cols...))
}

func (s stepDo) Omit(cols ...field.Expr) *stepDo {
	return s.withDO(s.DO.Omit(cols...))
}

func (s stepDo) Join(table schema.Tabler, on ...field.Expr) *stepDo {
	return s.withDO(s.DO.Join(table, on...))
}

func (s stepDo) LeftJoin(table schema.Tabler, on ...field.Expr) *stepDo {
	return s.withDO(s.DO.LeftJoin(table, on...))
}

func (s stepDo) RightJoin(table schema.Tabler, on ...field.Expr) *stepDo {
	return s.withDO(s.DO.RightJoin(table, on...))
}

func (s stepDo) Group(cols ...field.Expr) *stepDo {
	return s.withDO(s.DO.Group(cols...))
}

func (s stepDo) Having(conds ...gen.Condition) *stepDo {
	return s.withDO(s.DO.Having(conds...))
}

func (s stepDo) Limit(limit int) *stepDo {
	return s.withDO(s.DO.Limit(limit))
}

func (s stepDo) Offset(offset int) *stepDo {
	return s.withDO(s.DO.Offset(offset))
}

func (s stepDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *stepDo {
	return s.withDO(s.DO.Scopes(funcs...))
}

func (s stepDo) Unscoped() *stepDo {
	return s.withDO(s.DO.Unscoped())
}

func (s stepDo) Create(values ...*model.Step) error {
	if len(values) == 0 {
		return nil
	}
	return s.DO.Create(values)
}

func (s stepDo) CreateInBatches(values []*model.Step, batchSize int) error {
	return s.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (s stepDo) Save(values ...*model.Step) error {
	if len(values) == 0 {
		return nil
	}
	return s.DO.Save(values)
}

func (s stepDo) First() (*model.Step, error) {
	if result, err := s.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Step), nil
	}
}

func (s stepDo) Take() (*model.Step, error) {
	if result, err := s.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Step), nil
	}
}

func (s stepDo) Last() (*model.Step, error) {
	if result, err := s.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Step), nil
	}
}

func (s stepDo) Find() ([]*model.Step, error) {
	result, err := s.DO.Find()
	return result.([]*model.Step), err
}

func (s stepDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Step, err error) {
	buf := make([]*model.Step, 0, batchSize)
	err = s.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (s stepDo) FindInBatches(result *[]*model.Step, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return s.DO.FindInBatches(result, batchSize, fc)
}

func (s stepDo) Attrs(attrs ...field.AssignExpr) *stepDo {
	return s.withDO(s.DO.Attrs(attrs...))
}

func (s stepDo) Assign(attrs ...field.AssignExpr) *stepDo {
	return s.withDO(s.DO.Assign(attrs...))
}

func (s stepDo) Joins(fields ...field.RelationField) *stepDo {
	for _, _f := range fields {
		s = *s.withDO(s.DO.Joins(_f))
	}
	return &s
}

func (s stepDo) Preload(fields ...field.RelationField) *stepDo {
	for _, _f := range fields {
		s = *s.withDO(s.DO.Preload(_f))
	}
	return &s
}

func (s stepDo) FirstOrInit() (*model.Step, error) {
	if result, err := s.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Step), nil
	}
}

func (s stepDo) FirstOrCreate() (*model.Step, error) {
	if result, err := s.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Step), nil
	}
}

func (s stepDo) FindByPage(offset int, limit int) (result []*model.Step, count int64, err error) {
	result, err = s.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = s.Offset(-1).Limit(-1).Count()
	return
}

func (s stepDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = s.Count()
	if err != nil {
		return
	}

	err = s.Offset(offset).Limit(limit).Scan(result)
	return
}

func (s stepDo) Scan(result interface{}) (err error) {
	return s.DO.Scan(result)
}

func (s stepDo) Delete(models ...*model.Step) (result gen.ResultInfo, err error) {
	return s.DO.Delete(models)
}

func (s *stepDo) withDO(do gen.Dao) *stepDo {
	s.DO = *do.(*gen.DO)
	return s
}

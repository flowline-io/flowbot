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

func newPipeline(db *gorm.DB, opts ...gen.DOOption) pipeline {
	_pipeline := pipeline{}

	_pipeline.pipelineDo.UseDB(db, opts...)
	_pipeline.pipelineDo.UseModel(&model.Pipeline{})

	tableName := _pipeline.pipelineDo.TableName()
	_pipeline.ALL = field.NewAsterisk(tableName)
	_pipeline.ID = field.NewInt64(tableName, "id")
	_pipeline.UID = field.NewString(tableName, "uid")
	_pipeline.Topic = field.NewString(tableName, "topic")
	_pipeline.Flag = field.NewString(tableName, "flag")
	_pipeline.RuleID = field.NewString(tableName, "rule_id")
	_pipeline.Version = field.NewInt32(tableName, "version")
	_pipeline.Stage = field.NewInt32(tableName, "stage")
	_pipeline.Values = field.NewField(tableName, "values")
	_pipeline.State = field.NewField(tableName, "state")
	_pipeline.CreatedAt = field.NewTime(tableName, "created_at")
	_pipeline.UpdatedAt = field.NewTime(tableName, "updated_at")

	_pipeline.fillFieldMap()

	return _pipeline
}

type pipeline struct {
	pipelineDo

	ALL       field.Asterisk
	ID        field.Int64
	UID       field.String
	Topic     field.String
	Flag      field.String
	RuleID    field.String
	Version   field.Int32
	Stage     field.Int32
	Values    field.Field
	State     field.Field
	CreatedAt field.Time
	UpdatedAt field.Time

	fieldMap map[string]field.Expr
}

func (p pipeline) Table(newTableName string) *pipeline {
	p.pipelineDo.UseTable(newTableName)
	return p.updateTableName(newTableName)
}

func (p pipeline) As(alias string) *pipeline {
	p.pipelineDo.DO = *(p.pipelineDo.As(alias).(*gen.DO))
	return p.updateTableName(alias)
}

func (p *pipeline) updateTableName(table string) *pipeline {
	p.ALL = field.NewAsterisk(table)
	p.ID = field.NewInt64(table, "id")
	p.UID = field.NewString(table, "uid")
	p.Topic = field.NewString(table, "topic")
	p.Flag = field.NewString(table, "flag")
	p.RuleID = field.NewString(table, "rule_id")
	p.Version = field.NewInt32(table, "version")
	p.Stage = field.NewInt32(table, "stage")
	p.Values = field.NewField(table, "values")
	p.State = field.NewField(table, "state")
	p.CreatedAt = field.NewTime(table, "created_at")
	p.UpdatedAt = field.NewTime(table, "updated_at")

	p.fillFieldMap()

	return p
}

func (p *pipeline) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := p.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (p *pipeline) fillFieldMap() {
	p.fieldMap = make(map[string]field.Expr, 11)
	p.fieldMap["id"] = p.ID
	p.fieldMap["uid"] = p.UID
	p.fieldMap["topic"] = p.Topic
	p.fieldMap["flag"] = p.Flag
	p.fieldMap["rule_id"] = p.RuleID
	p.fieldMap["version"] = p.Version
	p.fieldMap["stage"] = p.Stage
	p.fieldMap["values"] = p.Values
	p.fieldMap["state"] = p.State
	p.fieldMap["created_at"] = p.CreatedAt
	p.fieldMap["updated_at"] = p.UpdatedAt
}

func (p pipeline) clone(db *gorm.DB) pipeline {
	p.pipelineDo.ReplaceConnPool(db.Statement.ConnPool)
	return p
}

func (p pipeline) replaceDB(db *gorm.DB) pipeline {
	p.pipelineDo.ReplaceDB(db)
	return p
}

type pipelineDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (p pipelineDo) GetByID(id int) (result model.Pipeline, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM pipelines WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = p.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (p pipelineDo) TakeByUidAndTopic(uid string, topic string) (result model.Pipeline, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = p.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (p pipelineDo) FindByUidAndTopic(uid string, topic string) (result []*model.Pipeline, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = p.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (p pipelineDo) Debug() *pipelineDo {
	return p.withDO(p.DO.Debug())
}

func (p pipelineDo) WithContext(ctx context.Context) *pipelineDo {
	return p.withDO(p.DO.WithContext(ctx))
}

func (p pipelineDo) ReadDB() *pipelineDo {
	return p.Clauses(dbresolver.Read)
}

func (p pipelineDo) WriteDB() *pipelineDo {
	return p.Clauses(dbresolver.Write)
}

func (p pipelineDo) Session(config *gorm.Session) *pipelineDo {
	return p.withDO(p.DO.Session(config))
}

func (p pipelineDo) Clauses(conds ...clause.Expression) *pipelineDo {
	return p.withDO(p.DO.Clauses(conds...))
}

func (p pipelineDo) Returning(value interface{}, columns ...string) *pipelineDo {
	return p.withDO(p.DO.Returning(value, columns...))
}

func (p pipelineDo) Not(conds ...gen.Condition) *pipelineDo {
	return p.withDO(p.DO.Not(conds...))
}

func (p pipelineDo) Or(conds ...gen.Condition) *pipelineDo {
	return p.withDO(p.DO.Or(conds...))
}

func (p pipelineDo) Select(conds ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Select(conds...))
}

func (p pipelineDo) Where(conds ...gen.Condition) *pipelineDo {
	return p.withDO(p.DO.Where(conds...))
}

func (p pipelineDo) Order(conds ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Order(conds...))
}

func (p pipelineDo) Distinct(cols ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Distinct(cols...))
}

func (p pipelineDo) Omit(cols ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Omit(cols...))
}

func (p pipelineDo) Join(table schema.Tabler, on ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Join(table, on...))
}

func (p pipelineDo) LeftJoin(table schema.Tabler, on ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.LeftJoin(table, on...))
}

func (p pipelineDo) RightJoin(table schema.Tabler, on ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.RightJoin(table, on...))
}

func (p pipelineDo) Group(cols ...field.Expr) *pipelineDo {
	return p.withDO(p.DO.Group(cols...))
}

func (p pipelineDo) Having(conds ...gen.Condition) *pipelineDo {
	return p.withDO(p.DO.Having(conds...))
}

func (p pipelineDo) Limit(limit int) *pipelineDo {
	return p.withDO(p.DO.Limit(limit))
}

func (p pipelineDo) Offset(offset int) *pipelineDo {
	return p.withDO(p.DO.Offset(offset))
}

func (p pipelineDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *pipelineDo {
	return p.withDO(p.DO.Scopes(funcs...))
}

func (p pipelineDo) Unscoped() *pipelineDo {
	return p.withDO(p.DO.Unscoped())
}

func (p pipelineDo) Create(values ...*model.Pipeline) error {
	if len(values) == 0 {
		return nil
	}
	return p.DO.Create(values)
}

func (p pipelineDo) CreateInBatches(values []*model.Pipeline, batchSize int) error {
	return p.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (p pipelineDo) Save(values ...*model.Pipeline) error {
	if len(values) == 0 {
		return nil
	}
	return p.DO.Save(values)
}

func (p pipelineDo) First() (*model.Pipeline, error) {
	if result, err := p.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Pipeline), nil
	}
}

func (p pipelineDo) Take() (*model.Pipeline, error) {
	if result, err := p.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Pipeline), nil
	}
}

func (p pipelineDo) Last() (*model.Pipeline, error) {
	if result, err := p.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Pipeline), nil
	}
}

func (p pipelineDo) Find() ([]*model.Pipeline, error) {
	result, err := p.DO.Find()
	return result.([]*model.Pipeline), err
}

func (p pipelineDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Pipeline, err error) {
	buf := make([]*model.Pipeline, 0, batchSize)
	err = p.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (p pipelineDo) FindInBatches(result *[]*model.Pipeline, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return p.DO.FindInBatches(result, batchSize, fc)
}

func (p pipelineDo) Attrs(attrs ...field.AssignExpr) *pipelineDo {
	return p.withDO(p.DO.Attrs(attrs...))
}

func (p pipelineDo) Assign(attrs ...field.AssignExpr) *pipelineDo {
	return p.withDO(p.DO.Assign(attrs...))
}

func (p pipelineDo) Joins(fields ...field.RelationField) *pipelineDo {
	for _, _f := range fields {
		p = *p.withDO(p.DO.Joins(_f))
	}
	return &p
}

func (p pipelineDo) Preload(fields ...field.RelationField) *pipelineDo {
	for _, _f := range fields {
		p = *p.withDO(p.DO.Preload(_f))
	}
	return &p
}

func (p pipelineDo) FirstOrInit() (*model.Pipeline, error) {
	if result, err := p.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Pipeline), nil
	}
}

func (p pipelineDo) FirstOrCreate() (*model.Pipeline, error) {
	if result, err := p.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Pipeline), nil
	}
}

func (p pipelineDo) FindByPage(offset int, limit int) (result []*model.Pipeline, count int64, err error) {
	result, err = p.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = p.Offset(-1).Limit(-1).Count()
	return
}

func (p pipelineDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = p.Count()
	if err != nil {
		return
	}

	err = p.Offset(offset).Limit(limit).Scan(result)
	return
}

func (p pipelineDo) Scan(result interface{}) (err error) {
	return p.DO.Scan(result)
}

func (p pipelineDo) Delete(models ...*model.Pipeline) (result gen.ResultInfo, err error) {
	return p.DO.Delete(models)
}

func (p *pipelineDo) withDO(do gen.Dao) *pipelineDo {
	p.DO = *do.(*gen.DO)
	return p
}

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

func newTopic(db *gorm.DB, opts ...gen.DOOption) topic {
	_topic := topic{}

	_topic.topicDo.UseDB(db, opts...)
	_topic.topicDo.UseModel(&model.Topic{})

	tableName := _topic.topicDo.TableName()
	_topic.ALL = field.NewAsterisk(tableName)
	_topic.ID = field.NewInt32(tableName, "id")
	_topic.Createdat = field.NewTime(tableName, "createdat")
	_topic.Updatedat = field.NewTime(tableName, "updatedat")
	_topic.State = field.NewInt32(tableName, "state")
	_topic.Stateat = field.NewTime(tableName, "stateat")
	_topic.Touchedat = field.NewTime(tableName, "touchedat")
	_topic.Name = field.NewString(tableName, "name")
	_topic.Usebt = field.NewInt32(tableName, "usebt")
	_topic.Owner = field.NewInt64(tableName, "owner")
	_topic.Access = field.NewField(tableName, "access")
	_topic.Seqid = field.NewInt32(tableName, "seqid")
	_topic.Delid = field.NewInt32(tableName, "delid")
	_topic.Public = field.NewField(tableName, "public")
	_topic.Trusted = field.NewField(tableName, "trusted")
	_topic.Tags = field.NewString(tableName, "tags")

	_topic.fillFieldMap()

	return _topic
}

type topic struct {
	topicDo

	ALL       field.Asterisk
	ID        field.Int32
	Createdat field.Time
	Updatedat field.Time
	State     field.Int32
	Stateat   field.Time
	Touchedat field.Time
	Name      field.String
	Usebt     field.Int32
	Owner     field.Int64
	Access    field.Field
	Seqid     field.Int32
	Delid     field.Int32
	Public    field.Field
	Trusted   field.Field
	Tags      field.String

	fieldMap map[string]field.Expr
}

func (t topic) Table(newTableName string) *topic {
	t.topicDo.UseTable(newTableName)
	return t.updateTableName(newTableName)
}

func (t topic) As(alias string) *topic {
	t.topicDo.DO = *(t.topicDo.As(alias).(*gen.DO))
	return t.updateTableName(alias)
}

func (t *topic) updateTableName(table string) *topic {
	t.ALL = field.NewAsterisk(table)
	t.ID = field.NewInt32(table, "id")
	t.Createdat = field.NewTime(table, "createdat")
	t.Updatedat = field.NewTime(table, "updatedat")
	t.State = field.NewInt32(table, "state")
	t.Stateat = field.NewTime(table, "stateat")
	t.Touchedat = field.NewTime(table, "touchedat")
	t.Name = field.NewString(table, "name")
	t.Usebt = field.NewInt32(table, "usebt")
	t.Owner = field.NewInt64(table, "owner")
	t.Access = field.NewField(table, "access")
	t.Seqid = field.NewInt32(table, "seqid")
	t.Delid = field.NewInt32(table, "delid")
	t.Public = field.NewField(table, "public")
	t.Trusted = field.NewField(table, "trusted")
	t.Tags = field.NewString(table, "tags")

	t.fillFieldMap()

	return t
}

func (t *topic) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := t.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (t *topic) fillFieldMap() {
	t.fieldMap = make(map[string]field.Expr, 17)
	t.fieldMap["id"] = t.ID
	t.fieldMap["createdat"] = t.Createdat
	t.fieldMap["updatedat"] = t.Updatedat
	t.fieldMap["state"] = t.State
	t.fieldMap["stateat"] = t.Stateat
	t.fieldMap["touchedat"] = t.Touchedat
	t.fieldMap["name"] = t.Name
	t.fieldMap["usebt"] = t.Usebt
	t.fieldMap["owner"] = t.Owner
	t.fieldMap["access"] = t.Access
	t.fieldMap["seqid"] = t.Seqid
	t.fieldMap["delid"] = t.Delid
	t.fieldMap["public"] = t.Public
	t.fieldMap["trusted"] = t.Trusted
	t.fieldMap["tags"] = t.Tags

}

func (t topic) clone(db *gorm.DB) topic {
	t.topicDo.ReplaceConnPool(db.Statement.ConnPool)
	return t
}

func (t topic) replaceDB(db *gorm.DB) topic {
	t.topicDo.ReplaceDB(db)
	return t
}

type topicDo struct{ gen.DO }

func (t topicDo) Debug() *topicDo {
	return t.withDO(t.DO.Debug())
}

func (t topicDo) WithContext(ctx context.Context) *topicDo {
	return t.withDO(t.DO.WithContext(ctx))
}

func (t topicDo) ReadDB() *topicDo {
	return t.Clauses(dbresolver.Read)
}

func (t topicDo) WriteDB() *topicDo {
	return t.Clauses(dbresolver.Write)
}

func (t topicDo) Session(config *gorm.Session) *topicDo {
	return t.withDO(t.DO.Session(config))
}

func (t topicDo) Clauses(conds ...clause.Expression) *topicDo {
	return t.withDO(t.DO.Clauses(conds...))
}

func (t topicDo) Returning(value interface{}, columns ...string) *topicDo {
	return t.withDO(t.DO.Returning(value, columns...))
}

func (t topicDo) Not(conds ...gen.Condition) *topicDo {
	return t.withDO(t.DO.Not(conds...))
}

func (t topicDo) Or(conds ...gen.Condition) *topicDo {
	return t.withDO(t.DO.Or(conds...))
}

func (t topicDo) Select(conds ...field.Expr) *topicDo {
	return t.withDO(t.DO.Select(conds...))
}

func (t topicDo) Where(conds ...gen.Condition) *topicDo {
	return t.withDO(t.DO.Where(conds...))
}

func (t topicDo) Order(conds ...field.Expr) *topicDo {
	return t.withDO(t.DO.Order(conds...))
}

func (t topicDo) Distinct(cols ...field.Expr) *topicDo {
	return t.withDO(t.DO.Distinct(cols...))
}

func (t topicDo) Omit(cols ...field.Expr) *topicDo {
	return t.withDO(t.DO.Omit(cols...))
}

func (t topicDo) Join(table schema.Tabler, on ...field.Expr) *topicDo {
	return t.withDO(t.DO.Join(table, on...))
}

func (t topicDo) LeftJoin(table schema.Tabler, on ...field.Expr) *topicDo {
	return t.withDO(t.DO.LeftJoin(table, on...))
}

func (t topicDo) RightJoin(table schema.Tabler, on ...field.Expr) *topicDo {
	return t.withDO(t.DO.RightJoin(table, on...))
}

func (t topicDo) Group(cols ...field.Expr) *topicDo {
	return t.withDO(t.DO.Group(cols...))
}

func (t topicDo) Having(conds ...gen.Condition) *topicDo {
	return t.withDO(t.DO.Having(conds...))
}

func (t topicDo) Limit(limit int) *topicDo {
	return t.withDO(t.DO.Limit(limit))
}

func (t topicDo) Offset(offset int) *topicDo {
	return t.withDO(t.DO.Offset(offset))
}

func (t topicDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *topicDo {
	return t.withDO(t.DO.Scopes(funcs...))
}

func (t topicDo) Unscoped() *topicDo {
	return t.withDO(t.DO.Unscoped())
}

func (t topicDo) Create(values ...*model.Topic) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Create(values)
}

func (t topicDo) CreateInBatches(values []*model.Topic, batchSize int) error {
	return t.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (t topicDo) Save(values ...*model.Topic) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Save(values)
}

func (t topicDo) First() (*model.Topic, error) {
	if result, err := t.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topic), nil
	}
}

func (t topicDo) Take() (*model.Topic, error) {
	if result, err := t.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topic), nil
	}
}

func (t topicDo) Last() (*model.Topic, error) {
	if result, err := t.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topic), nil
	}
}

func (t topicDo) Find() ([]*model.Topic, error) {
	result, err := t.DO.Find()
	return result.([]*model.Topic), err
}

func (t topicDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Topic, err error) {
	buf := make([]*model.Topic, 0, batchSize)
	err = t.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (t topicDo) FindInBatches(result *[]*model.Topic, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return t.DO.FindInBatches(result, batchSize, fc)
}

func (t topicDo) Attrs(attrs ...field.AssignExpr) *topicDo {
	return t.withDO(t.DO.Attrs(attrs...))
}

func (t topicDo) Assign(attrs ...field.AssignExpr) *topicDo {
	return t.withDO(t.DO.Assign(attrs...))
}

func (t topicDo) Joins(fields ...field.RelationField) *topicDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Joins(_f))
	}
	return &t
}

func (t topicDo) Preload(fields ...field.RelationField) *topicDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Preload(_f))
	}
	return &t
}

func (t topicDo) FirstOrInit() (*model.Topic, error) {
	if result, err := t.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topic), nil
	}
}

func (t topicDo) FirstOrCreate() (*model.Topic, error) {
	if result, err := t.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topic), nil
	}
}

func (t topicDo) FindByPage(offset int, limit int) (result []*model.Topic, count int64, err error) {
	result, err = t.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = t.Offset(-1).Limit(-1).Count()
	return
}

func (t topicDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = t.Count()
	if err != nil {
		return
	}

	err = t.Offset(offset).Limit(limit).Scan(result)
	return
}

func (t topicDo) Scan(result interface{}) (err error) {
	return t.DO.Scan(result)
}

func (t topicDo) Delete(models ...*model.Topic) (result gen.ResultInfo, err error) {
	return t.DO.Delete(models)
}

func (t *topicDo) withDO(do gen.Dao) *topicDo {
	t.DO = *do.(*gen.DO)
	return t
}

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

	"github.com/sysatom/flowbot/internal/store/model"
)

func newTopictag(db *gorm.DB, opts ...gen.DOOption) topictag {
	_topictag := topictag{}

	_topictag.topictagDo.UseDB(db, opts...)
	_topictag.topictagDo.UseModel(&model.Topictag{})

	tableName := _topictag.topictagDo.TableName()
	_topictag.ALL = field.NewAsterisk(tableName)
	_topictag.ID = field.NewInt32(tableName, "id")
	_topictag.Topic = field.NewString(tableName, "topic")
	_topictag.Tag = field.NewString(tableName, "tag")

	_topictag.fillFieldMap()

	return _topictag
}

type topictag struct {
	topictagDo

	ALL   field.Asterisk
	ID    field.Int32
	Topic field.String
	Tag   field.String

	fieldMap map[string]field.Expr
}

func (t topictag) Table(newTableName string) *topictag {
	t.topictagDo.UseTable(newTableName)
	return t.updateTableName(newTableName)
}

func (t topictag) As(alias string) *topictag {
	t.topictagDo.DO = *(t.topictagDo.As(alias).(*gen.DO))
	return t.updateTableName(alias)
}

func (t *topictag) updateTableName(table string) *topictag {
	t.ALL = field.NewAsterisk(table)
	t.ID = field.NewInt32(table, "id")
	t.Topic = field.NewString(table, "topic")
	t.Tag = field.NewString(table, "tag")

	t.fillFieldMap()

	return t
}

func (t *topictag) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := t.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (t *topictag) fillFieldMap() {
	t.fieldMap = make(map[string]field.Expr, 3)
	t.fieldMap["id"] = t.ID
	t.fieldMap["topic"] = t.Topic
	t.fieldMap["tag"] = t.Tag
}

func (t topictag) clone(db *gorm.DB) topictag {
	t.topictagDo.ReplaceConnPool(db.Statement.ConnPool)
	return t
}

func (t topictag) replaceDB(db *gorm.DB) topictag {
	t.topictagDo.ReplaceDB(db)
	return t
}

type topictagDo struct{ gen.DO }

func (t topictagDo) Debug() *topictagDo {
	return t.withDO(t.DO.Debug())
}

func (t topictagDo) WithContext(ctx context.Context) *topictagDo {
	return t.withDO(t.DO.WithContext(ctx))
}

func (t topictagDo) ReadDB() *topictagDo {
	return t.Clauses(dbresolver.Read)
}

func (t topictagDo) WriteDB() *topictagDo {
	return t.Clauses(dbresolver.Write)
}

func (t topictagDo) Session(config *gorm.Session) *topictagDo {
	return t.withDO(t.DO.Session(config))
}

func (t topictagDo) Clauses(conds ...clause.Expression) *topictagDo {
	return t.withDO(t.DO.Clauses(conds...))
}

func (t topictagDo) Returning(value interface{}, columns ...string) *topictagDo {
	return t.withDO(t.DO.Returning(value, columns...))
}

func (t topictagDo) Not(conds ...gen.Condition) *topictagDo {
	return t.withDO(t.DO.Not(conds...))
}

func (t topictagDo) Or(conds ...gen.Condition) *topictagDo {
	return t.withDO(t.DO.Or(conds...))
}

func (t topictagDo) Select(conds ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Select(conds...))
}

func (t topictagDo) Where(conds ...gen.Condition) *topictagDo {
	return t.withDO(t.DO.Where(conds...))
}

func (t topictagDo) Order(conds ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Order(conds...))
}

func (t topictagDo) Distinct(cols ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Distinct(cols...))
}

func (t topictagDo) Omit(cols ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Omit(cols...))
}

func (t topictagDo) Join(table schema.Tabler, on ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Join(table, on...))
}

func (t topictagDo) LeftJoin(table schema.Tabler, on ...field.Expr) *topictagDo {
	return t.withDO(t.DO.LeftJoin(table, on...))
}

func (t topictagDo) RightJoin(table schema.Tabler, on ...field.Expr) *topictagDo {
	return t.withDO(t.DO.RightJoin(table, on...))
}

func (t topictagDo) Group(cols ...field.Expr) *topictagDo {
	return t.withDO(t.DO.Group(cols...))
}

func (t topictagDo) Having(conds ...gen.Condition) *topictagDo {
	return t.withDO(t.DO.Having(conds...))
}

func (t topictagDo) Limit(limit int) *topictagDo {
	return t.withDO(t.DO.Limit(limit))
}

func (t topictagDo) Offset(offset int) *topictagDo {
	return t.withDO(t.DO.Offset(offset))
}

func (t topictagDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *topictagDo {
	return t.withDO(t.DO.Scopes(funcs...))
}

func (t topictagDo) Unscoped() *topictagDo {
	return t.withDO(t.DO.Unscoped())
}

func (t topictagDo) Create(values ...*model.Topictag) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Create(values)
}

func (t topictagDo) CreateInBatches(values []*model.Topictag, batchSize int) error {
	return t.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (t topictagDo) Save(values ...*model.Topictag) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Save(values)
}

func (t topictagDo) First() (*model.Topictag, error) {
	if result, err := t.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topictag), nil
	}
}

func (t topictagDo) Take() (*model.Topictag, error) {
	if result, err := t.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topictag), nil
	}
}

func (t topictagDo) Last() (*model.Topictag, error) {
	if result, err := t.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topictag), nil
	}
}

func (t topictagDo) Find() ([]*model.Topictag, error) {
	result, err := t.DO.Find()
	return result.([]*model.Topictag), err
}

func (t topictagDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Topictag, err error) {
	buf := make([]*model.Topictag, 0, batchSize)
	err = t.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (t topictagDo) FindInBatches(result *[]*model.Topictag, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return t.DO.FindInBatches(result, batchSize, fc)
}

func (t topictagDo) Attrs(attrs ...field.AssignExpr) *topictagDo {
	return t.withDO(t.DO.Attrs(attrs...))
}

func (t topictagDo) Assign(attrs ...field.AssignExpr) *topictagDo {
	return t.withDO(t.DO.Assign(attrs...))
}

func (t topictagDo) Joins(fields ...field.RelationField) *topictagDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Joins(_f))
	}
	return &t
}

func (t topictagDo) Preload(fields ...field.RelationField) *topictagDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Preload(_f))
	}
	return &t
}

func (t topictagDo) FirstOrInit() (*model.Topictag, error) {
	if result, err := t.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topictag), nil
	}
}

func (t topictagDo) FirstOrCreate() (*model.Topictag, error) {
	if result, err := t.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Topictag), nil
	}
}

func (t topictagDo) FindByPage(offset int, limit int) (result []*model.Topictag, count int64, err error) {
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

func (t topictagDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = t.Count()
	if err != nil {
		return
	}

	err = t.Offset(offset).Limit(limit).Scan(result)
	return
}

func (t topictagDo) Scan(result interface{}) (err error) {
	return t.DO.Scan(result)
}

func (t topictagDo) Delete(models ...*model.Topictag) (result gen.ResultInfo, err error) {
	return t.DO.Delete(models)
}

func (t *topictagDo) withDO(do gen.Dao) *topictagDo {
	t.DO = *do.(*gen.DO)
	return t
}

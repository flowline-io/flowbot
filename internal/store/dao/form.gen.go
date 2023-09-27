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

func newForm(db *gorm.DB, opts ...gen.DOOption) form {
	_form := form{}

	_form.formDo.UseDB(db, opts...)
	_form.formDo.UseModel(&model.Form{})

	tableName := _form.formDo.TableName()
	_form.ALL = field.NewAsterisk(tableName)
	_form.ID = field.NewInt64(tableName, "id")
	_form.FormID = field.NewString(tableName, "form_id")
	_form.UID = field.NewString(tableName, "uid")
	_form.Topic = field.NewString(tableName, "topic")
	_form.Schema = field.NewField(tableName, "schema")
	_form.Values = field.NewField(tableName, "values")
	_form.Extra = field.NewField(tableName, "extra")
	_form.State = field.NewField(tableName, "state")
	_form.CreatedAt = field.NewTime(tableName, "created_at")
	_form.UpdatedAt = field.NewTime(tableName, "updated_at")

	_form.fillFieldMap()

	return _form
}

type form struct {
	formDo

	ALL       field.Asterisk
	ID        field.Int64
	FormID    field.String
	UID       field.String
	Topic     field.String
	Schema    field.Field
	Values    field.Field
	Extra     field.Field
	State     field.Field
	CreatedAt field.Time
	UpdatedAt field.Time

	fieldMap map[string]field.Expr
}

func (f form) Table(newTableName string) *form {
	f.formDo.UseTable(newTableName)
	return f.updateTableName(newTableName)
}

func (f form) As(alias string) *form {
	f.formDo.DO = *(f.formDo.As(alias).(*gen.DO))
	return f.updateTableName(alias)
}

func (f *form) updateTableName(table string) *form {
	f.ALL = field.NewAsterisk(table)
	f.ID = field.NewInt64(table, "id")
	f.FormID = field.NewString(table, "form_id")
	f.UID = field.NewString(table, "uid")
	f.Topic = field.NewString(table, "topic")
	f.Schema = field.NewField(table, "schema")
	f.Values = field.NewField(table, "values")
	f.Extra = field.NewField(table, "extra")
	f.State = field.NewField(table, "state")
	f.CreatedAt = field.NewTime(table, "created_at")
	f.UpdatedAt = field.NewTime(table, "updated_at")

	f.fillFieldMap()

	return f
}

func (f *form) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := f.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (f *form) fillFieldMap() {
	f.fieldMap = make(map[string]field.Expr, 10)
	f.fieldMap["id"] = f.ID
	f.fieldMap["form_id"] = f.FormID
	f.fieldMap["uid"] = f.UID
	f.fieldMap["topic"] = f.Topic
	f.fieldMap["schema"] = f.Schema
	f.fieldMap["values"] = f.Values
	f.fieldMap["extra"] = f.Extra
	f.fieldMap["state"] = f.State
	f.fieldMap["created_at"] = f.CreatedAt
	f.fieldMap["updated_at"] = f.UpdatedAt
}

func (f form) clone(db *gorm.DB) form {
	f.formDo.ReplaceConnPool(db.Statement.ConnPool)
	return f
}

func (f form) replaceDB(db *gorm.DB) form {
	f.formDo.ReplaceDB(db)
	return f
}

type formDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (f formDo) GetByID(id int) (result model.Form, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM form WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = f.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (f formDo) TakeByUidAndTopic(uid string, topic string) (result model.Form, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = f.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (f formDo) FindByUidAndTopic(uid string, topic string) (result []*model.Form, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = f.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (f formDo) Debug() *formDo {
	return f.withDO(f.DO.Debug())
}

func (f formDo) WithContext(ctx context.Context) *formDo {
	return f.withDO(f.DO.WithContext(ctx))
}

func (f formDo) ReadDB() *formDo {
	return f.Clauses(dbresolver.Read)
}

func (f formDo) WriteDB() *formDo {
	return f.Clauses(dbresolver.Write)
}

func (f formDo) Session(config *gorm.Session) *formDo {
	return f.withDO(f.DO.Session(config))
}

func (f formDo) Clauses(conds ...clause.Expression) *formDo {
	return f.withDO(f.DO.Clauses(conds...))
}

func (f formDo) Returning(value interface{}, columns ...string) *formDo {
	return f.withDO(f.DO.Returning(value, columns...))
}

func (f formDo) Not(conds ...gen.Condition) *formDo {
	return f.withDO(f.DO.Not(conds...))
}

func (f formDo) Or(conds ...gen.Condition) *formDo {
	return f.withDO(f.DO.Or(conds...))
}

func (f formDo) Select(conds ...field.Expr) *formDo {
	return f.withDO(f.DO.Select(conds...))
}

func (f formDo) Where(conds ...gen.Condition) *formDo {
	return f.withDO(f.DO.Where(conds...))
}

func (f formDo) Order(conds ...field.Expr) *formDo {
	return f.withDO(f.DO.Order(conds...))
}

func (f formDo) Distinct(cols ...field.Expr) *formDo {
	return f.withDO(f.DO.Distinct(cols...))
}

func (f formDo) Omit(cols ...field.Expr) *formDo {
	return f.withDO(f.DO.Omit(cols...))
}

func (f formDo) Join(table schema.Tabler, on ...field.Expr) *formDo {
	return f.withDO(f.DO.Join(table, on...))
}

func (f formDo) LeftJoin(table schema.Tabler, on ...field.Expr) *formDo {
	return f.withDO(f.DO.LeftJoin(table, on...))
}

func (f formDo) RightJoin(table schema.Tabler, on ...field.Expr) *formDo {
	return f.withDO(f.DO.RightJoin(table, on...))
}

func (f formDo) Group(cols ...field.Expr) *formDo {
	return f.withDO(f.DO.Group(cols...))
}

func (f formDo) Having(conds ...gen.Condition) *formDo {
	return f.withDO(f.DO.Having(conds...))
}

func (f formDo) Limit(limit int) *formDo {
	return f.withDO(f.DO.Limit(limit))
}

func (f formDo) Offset(offset int) *formDo {
	return f.withDO(f.DO.Offset(offset))
}

func (f formDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *formDo {
	return f.withDO(f.DO.Scopes(funcs...))
}

func (f formDo) Unscoped() *formDo {
	return f.withDO(f.DO.Unscoped())
}

func (f formDo) Create(values ...*model.Form) error {
	if len(values) == 0 {
		return nil
	}
	return f.DO.Create(values)
}

func (f formDo) CreateInBatches(values []*model.Form, batchSize int) error {
	return f.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (f formDo) Save(values ...*model.Form) error {
	if len(values) == 0 {
		return nil
	}
	return f.DO.Save(values)
}

func (f formDo) First() (*model.Form, error) {
	if result, err := f.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Form), nil
	}
}

func (f formDo) Take() (*model.Form, error) {
	if result, err := f.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Form), nil
	}
}

func (f formDo) Last() (*model.Form, error) {
	if result, err := f.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Form), nil
	}
}

func (f formDo) Find() ([]*model.Form, error) {
	result, err := f.DO.Find()
	return result.([]*model.Form), err
}

func (f formDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Form, err error) {
	buf := make([]*model.Form, 0, batchSize)
	err = f.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (f formDo) FindInBatches(result *[]*model.Form, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return f.DO.FindInBatches(result, batchSize, fc)
}

func (f formDo) Attrs(attrs ...field.AssignExpr) *formDo {
	return f.withDO(f.DO.Attrs(attrs...))
}

func (f formDo) Assign(attrs ...field.AssignExpr) *formDo {
	return f.withDO(f.DO.Assign(attrs...))
}

func (f formDo) Joins(fields ...field.RelationField) *formDo {
	for _, _f := range fields {
		f = *f.withDO(f.DO.Joins(_f))
	}
	return &f
}

func (f formDo) Preload(fields ...field.RelationField) *formDo {
	for _, _f := range fields {
		f = *f.withDO(f.DO.Preload(_f))
	}
	return &f
}

func (f formDo) FirstOrInit() (*model.Form, error) {
	if result, err := f.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Form), nil
	}
}

func (f formDo) FirstOrCreate() (*model.Form, error) {
	if result, err := f.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Form), nil
	}
}

func (f formDo) FindByPage(offset int, limit int) (result []*model.Form, count int64, err error) {
	result, err = f.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = f.Offset(-1).Limit(-1).Count()
	return
}

func (f formDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = f.Count()
	if err != nil {
		return
	}

	err = f.Offset(offset).Limit(limit).Scan(result)
	return
}

func (f formDo) Scan(result interface{}) (err error) {
	return f.DO.Scan(result)
}

func (f formDo) Delete(models ...*model.Form) (result gen.ResultInfo, err error) {
	return f.DO.Delete(models)
}

func (f *formDo) withDO(do gen.Dao) *formDo {
	f.DO = *do.(*gen.DO)
	return f
}

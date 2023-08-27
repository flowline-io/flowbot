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

	"github.com/sysatom/flowbot/internal/store/model"
)

func newKeyResultValue(db *gorm.DB, opts ...gen.DOOption) keyResultValue {
	_keyResultValue := keyResultValue{}

	_keyResultValue.keyResultValueDo.UseDB(db, opts...)
	_keyResultValue.keyResultValueDo.UseModel(&model.KeyResultValue{})

	tableName := _keyResultValue.keyResultValueDo.TableName()
	_keyResultValue.ALL = field.NewAsterisk(tableName)
	_keyResultValue.ID = field.NewInt32(tableName, "id")
	_keyResultValue.KeyResultID = field.NewInt32(tableName, "key_result_id")
	_keyResultValue.Value = field.NewInt32(tableName, "value")
	_keyResultValue.CreatedAt = field.NewTime(tableName, "created_at")
	_keyResultValue.UpdatedAt = field.NewTime(tableName, "updated_at")

	_keyResultValue.fillFieldMap()

	return _keyResultValue
}

type keyResultValue struct {
	keyResultValueDo

	ALL         field.Asterisk
	ID          field.Int32
	KeyResultID field.Int32
	Value       field.Int32
	CreatedAt   field.Time
	UpdatedAt   field.Time

	fieldMap map[string]field.Expr
}

func (k keyResultValue) Table(newTableName string) *keyResultValue {
	k.keyResultValueDo.UseTable(newTableName)
	return k.updateTableName(newTableName)
}

func (k keyResultValue) As(alias string) *keyResultValue {
	k.keyResultValueDo.DO = *(k.keyResultValueDo.As(alias).(*gen.DO))
	return k.updateTableName(alias)
}

func (k *keyResultValue) updateTableName(table string) *keyResultValue {
	k.ALL = field.NewAsterisk(table)
	k.ID = field.NewInt32(table, "id")
	k.KeyResultID = field.NewInt32(table, "key_result_id")
	k.Value = field.NewInt32(table, "value")
	k.CreatedAt = field.NewTime(table, "created_at")
	k.UpdatedAt = field.NewTime(table, "updated_at")

	k.fillFieldMap()

	return k
}

func (k *keyResultValue) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := k.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (k *keyResultValue) fillFieldMap() {
	k.fieldMap = make(map[string]field.Expr, 5)
	k.fieldMap["id"] = k.ID
	k.fieldMap["key_result_id"] = k.KeyResultID
	k.fieldMap["value"] = k.Value
	k.fieldMap["created_at"] = k.CreatedAt
	k.fieldMap["updated_at"] = k.UpdatedAt
}

func (k keyResultValue) clone(db *gorm.DB) keyResultValue {
	k.keyResultValueDo.ReplaceConnPool(db.Statement.ConnPool)
	return k
}

func (k keyResultValue) replaceDB(db *gorm.DB) keyResultValue {
	k.keyResultValueDo.ReplaceDB(db)
	return k
}

type keyResultValueDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (k keyResultValueDo) GetByID(id int) (result model.KeyResultValue, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM chatbot_key_result_values WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = k.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (k keyResultValueDo) TakeByUidAndTopic(uid string, topic string) (result model.KeyResultValue, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = k.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (k keyResultValueDo) FindByUidAndTopic(uid string, topic string) (result []*model.KeyResultValue, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = k.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (k keyResultValueDo) Debug() *keyResultValueDo {
	return k.withDO(k.DO.Debug())
}

func (k keyResultValueDo) WithContext(ctx context.Context) *keyResultValueDo {
	return k.withDO(k.DO.WithContext(ctx))
}

func (k keyResultValueDo) ReadDB() *keyResultValueDo {
	return k.Clauses(dbresolver.Read)
}

func (k keyResultValueDo) WriteDB() *keyResultValueDo {
	return k.Clauses(dbresolver.Write)
}

func (k keyResultValueDo) Session(config *gorm.Session) *keyResultValueDo {
	return k.withDO(k.DO.Session(config))
}

func (k keyResultValueDo) Clauses(conds ...clause.Expression) *keyResultValueDo {
	return k.withDO(k.DO.Clauses(conds...))
}

func (k keyResultValueDo) Returning(value interface{}, columns ...string) *keyResultValueDo {
	return k.withDO(k.DO.Returning(value, columns...))
}

func (k keyResultValueDo) Not(conds ...gen.Condition) *keyResultValueDo {
	return k.withDO(k.DO.Not(conds...))
}

func (k keyResultValueDo) Or(conds ...gen.Condition) *keyResultValueDo {
	return k.withDO(k.DO.Or(conds...))
}

func (k keyResultValueDo) Select(conds ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Select(conds...))
}

func (k keyResultValueDo) Where(conds ...gen.Condition) *keyResultValueDo {
	return k.withDO(k.DO.Where(conds...))
}

func (k keyResultValueDo) Order(conds ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Order(conds...))
}

func (k keyResultValueDo) Distinct(cols ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Distinct(cols...))
}

func (k keyResultValueDo) Omit(cols ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Omit(cols...))
}

func (k keyResultValueDo) Join(table schema.Tabler, on ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Join(table, on...))
}

func (k keyResultValueDo) LeftJoin(table schema.Tabler, on ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.LeftJoin(table, on...))
}

func (k keyResultValueDo) RightJoin(table schema.Tabler, on ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.RightJoin(table, on...))
}

func (k keyResultValueDo) Group(cols ...field.Expr) *keyResultValueDo {
	return k.withDO(k.DO.Group(cols...))
}

func (k keyResultValueDo) Having(conds ...gen.Condition) *keyResultValueDo {
	return k.withDO(k.DO.Having(conds...))
}

func (k keyResultValueDo) Limit(limit int) *keyResultValueDo {
	return k.withDO(k.DO.Limit(limit))
}

func (k keyResultValueDo) Offset(offset int) *keyResultValueDo {
	return k.withDO(k.DO.Offset(offset))
}

func (k keyResultValueDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *keyResultValueDo {
	return k.withDO(k.DO.Scopes(funcs...))
}

func (k keyResultValueDo) Unscoped() *keyResultValueDo {
	return k.withDO(k.DO.Unscoped())
}

func (k keyResultValueDo) Create(values ...*model.KeyResultValue) error {
	if len(values) == 0 {
		return nil
	}
	return k.DO.Create(values)
}

func (k keyResultValueDo) CreateInBatches(values []*model.KeyResultValue, batchSize int) error {
	return k.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (k keyResultValueDo) Save(values ...*model.KeyResultValue) error {
	if len(values) == 0 {
		return nil
	}
	return k.DO.Save(values)
}

func (k keyResultValueDo) First() (*model.KeyResultValue, error) {
	if result, err := k.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.KeyResultValue), nil
	}
}

func (k keyResultValueDo) Take() (*model.KeyResultValue, error) {
	if result, err := k.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.KeyResultValue), nil
	}
}

func (k keyResultValueDo) Last() (*model.KeyResultValue, error) {
	if result, err := k.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.KeyResultValue), nil
	}
}

func (k keyResultValueDo) Find() ([]*model.KeyResultValue, error) {
	result, err := k.DO.Find()
	return result.([]*model.KeyResultValue), err
}

func (k keyResultValueDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.KeyResultValue, err error) {
	buf := make([]*model.KeyResultValue, 0, batchSize)
	err = k.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (k keyResultValueDo) FindInBatches(result *[]*model.KeyResultValue, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return k.DO.FindInBatches(result, batchSize, fc)
}

func (k keyResultValueDo) Attrs(attrs ...field.AssignExpr) *keyResultValueDo {
	return k.withDO(k.DO.Attrs(attrs...))
}

func (k keyResultValueDo) Assign(attrs ...field.AssignExpr) *keyResultValueDo {
	return k.withDO(k.DO.Assign(attrs...))
}

func (k keyResultValueDo) Joins(fields ...field.RelationField) *keyResultValueDo {
	for _, _f := range fields {
		k = *k.withDO(k.DO.Joins(_f))
	}
	return &k
}

func (k keyResultValueDo) Preload(fields ...field.RelationField) *keyResultValueDo {
	for _, _f := range fields {
		k = *k.withDO(k.DO.Preload(_f))
	}
	return &k
}

func (k keyResultValueDo) FirstOrInit() (*model.KeyResultValue, error) {
	if result, err := k.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.KeyResultValue), nil
	}
}

func (k keyResultValueDo) FirstOrCreate() (*model.KeyResultValue, error) {
	if result, err := k.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.KeyResultValue), nil
	}
}

func (k keyResultValueDo) FindByPage(offset int, limit int) (result []*model.KeyResultValue, count int64, err error) {
	result, err = k.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = k.Offset(-1).Limit(-1).Count()
	return
}

func (k keyResultValueDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = k.Count()
	if err != nil {
		return
	}

	err = k.Offset(offset).Limit(limit).Scan(result)
	return
}

func (k keyResultValueDo) Scan(result interface{}) (err error) {
	return k.DO.Scan(result)
}

func (k keyResultValueDo) Delete(models ...*model.KeyResultValue) (result gen.ResultInfo, err error) {
	return k.DO.Delete(models)
}

func (k *keyResultValueDo) withDO(do gen.Dao) *keyResultValueDo {
	k.DO = *do.(*gen.DO)
	return k
}
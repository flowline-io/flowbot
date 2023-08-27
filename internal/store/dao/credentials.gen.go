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

func newCredential(db *gorm.DB, opts ...gen.DOOption) credential {
	_credential := credential{}

	_credential.credentialDo.UseDB(db, opts...)
	_credential.credentialDo.UseModel(&model.Credential{})

	tableName := _credential.credentialDo.TableName()
	_credential.ALL = field.NewAsterisk(tableName)
	_credential.ID = field.NewInt32(tableName, "id")
	_credential.Createdat = field.NewTime(tableName, "createdat")
	_credential.Updatedat = field.NewTime(tableName, "updatedat")
	_credential.Deletedat = field.NewTime(tableName, "deletedat")
	_credential.Method = field.NewString(tableName, "method")
	_credential.Value = field.NewString(tableName, "value")
	_credential.Synthetic = field.NewString(tableName, "synthetic")
	_credential.Userid = field.NewInt64(tableName, "userid")
	_credential.Resp = field.NewString(tableName, "resp")
	_credential.Done = field.NewInt32(tableName, "done")
	_credential.Retries = field.NewInt32(tableName, "retries")

	_credential.fillFieldMap()

	return _credential
}

type credential struct {
	credentialDo

	ALL       field.Asterisk
	ID        field.Int32
	Createdat field.Time
	Updatedat field.Time
	Deletedat field.Time
	Method    field.String
	Value     field.String
	Synthetic field.String
	Userid    field.Int64
	Resp      field.String
	Done      field.Int32
	Retries   field.Int32

	fieldMap map[string]field.Expr
}

func (c credential) Table(newTableName string) *credential {
	c.credentialDo.UseTable(newTableName)
	return c.updateTableName(newTableName)
}

func (c credential) As(alias string) *credential {
	c.credentialDo.DO = *(c.credentialDo.As(alias).(*gen.DO))
	return c.updateTableName(alias)
}

func (c *credential) updateTableName(table string) *credential {
	c.ALL = field.NewAsterisk(table)
	c.ID = field.NewInt32(table, "id")
	c.Createdat = field.NewTime(table, "createdat")
	c.Updatedat = field.NewTime(table, "updatedat")
	c.Deletedat = field.NewTime(table, "deletedat")
	c.Method = field.NewString(table, "method")
	c.Value = field.NewString(table, "value")
	c.Synthetic = field.NewString(table, "synthetic")
	c.Userid = field.NewInt64(table, "userid")
	c.Resp = field.NewString(table, "resp")
	c.Done = field.NewInt32(table, "done")
	c.Retries = field.NewInt32(table, "retries")

	c.fillFieldMap()

	return c
}

func (c *credential) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := c.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (c *credential) fillFieldMap() {
	c.fieldMap = make(map[string]field.Expr, 11)
	c.fieldMap["id"] = c.ID
	c.fieldMap["createdat"] = c.Createdat
	c.fieldMap["updatedat"] = c.Updatedat
	c.fieldMap["deletedat"] = c.Deletedat
	c.fieldMap["method"] = c.Method
	c.fieldMap["value"] = c.Value
	c.fieldMap["synthetic"] = c.Synthetic
	c.fieldMap["userid"] = c.Userid
	c.fieldMap["resp"] = c.Resp
	c.fieldMap["done"] = c.Done
	c.fieldMap["retries"] = c.Retries
}

func (c credential) clone(db *gorm.DB) credential {
	c.credentialDo.ReplaceConnPool(db.Statement.ConnPool)
	return c
}

func (c credential) replaceDB(db *gorm.DB) credential {
	c.credentialDo.ReplaceDB(db)
	return c
}

type credentialDo struct{ gen.DO }

func (c credentialDo) Debug() *credentialDo {
	return c.withDO(c.DO.Debug())
}

func (c credentialDo) WithContext(ctx context.Context) *credentialDo {
	return c.withDO(c.DO.WithContext(ctx))
}

func (c credentialDo) ReadDB() *credentialDo {
	return c.Clauses(dbresolver.Read)
}

func (c credentialDo) WriteDB() *credentialDo {
	return c.Clauses(dbresolver.Write)
}

func (c credentialDo) Session(config *gorm.Session) *credentialDo {
	return c.withDO(c.DO.Session(config))
}

func (c credentialDo) Clauses(conds ...clause.Expression) *credentialDo {
	return c.withDO(c.DO.Clauses(conds...))
}

func (c credentialDo) Returning(value interface{}, columns ...string) *credentialDo {
	return c.withDO(c.DO.Returning(value, columns...))
}

func (c credentialDo) Not(conds ...gen.Condition) *credentialDo {
	return c.withDO(c.DO.Not(conds...))
}

func (c credentialDo) Or(conds ...gen.Condition) *credentialDo {
	return c.withDO(c.DO.Or(conds...))
}

func (c credentialDo) Select(conds ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Select(conds...))
}

func (c credentialDo) Where(conds ...gen.Condition) *credentialDo {
	return c.withDO(c.DO.Where(conds...))
}

func (c credentialDo) Order(conds ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Order(conds...))
}

func (c credentialDo) Distinct(cols ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Distinct(cols...))
}

func (c credentialDo) Omit(cols ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Omit(cols...))
}

func (c credentialDo) Join(table schema.Tabler, on ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Join(table, on...))
}

func (c credentialDo) LeftJoin(table schema.Tabler, on ...field.Expr) *credentialDo {
	return c.withDO(c.DO.LeftJoin(table, on...))
}

func (c credentialDo) RightJoin(table schema.Tabler, on ...field.Expr) *credentialDo {
	return c.withDO(c.DO.RightJoin(table, on...))
}

func (c credentialDo) Group(cols ...field.Expr) *credentialDo {
	return c.withDO(c.DO.Group(cols...))
}

func (c credentialDo) Having(conds ...gen.Condition) *credentialDo {
	return c.withDO(c.DO.Having(conds...))
}

func (c credentialDo) Limit(limit int) *credentialDo {
	return c.withDO(c.DO.Limit(limit))
}

func (c credentialDo) Offset(offset int) *credentialDo {
	return c.withDO(c.DO.Offset(offset))
}

func (c credentialDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *credentialDo {
	return c.withDO(c.DO.Scopes(funcs...))
}

func (c credentialDo) Unscoped() *credentialDo {
	return c.withDO(c.DO.Unscoped())
}

func (c credentialDo) Create(values ...*model.Credential) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Create(values)
}

func (c credentialDo) CreateInBatches(values []*model.Credential, batchSize int) error {
	return c.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (c credentialDo) Save(values ...*model.Credential) error {
	if len(values) == 0 {
		return nil
	}
	return c.DO.Save(values)
}

func (c credentialDo) First() (*model.Credential, error) {
	if result, err := c.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Credential), nil
	}
}

func (c credentialDo) Take() (*model.Credential, error) {
	if result, err := c.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Credential), nil
	}
}

func (c credentialDo) Last() (*model.Credential, error) {
	if result, err := c.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Credential), nil
	}
}

func (c credentialDo) Find() ([]*model.Credential, error) {
	result, err := c.DO.Find()
	return result.([]*model.Credential), err
}

func (c credentialDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Credential, err error) {
	buf := make([]*model.Credential, 0, batchSize)
	err = c.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (c credentialDo) FindInBatches(result *[]*model.Credential, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return c.DO.FindInBatches(result, batchSize, fc)
}

func (c credentialDo) Attrs(attrs ...field.AssignExpr) *credentialDo {
	return c.withDO(c.DO.Attrs(attrs...))
}

func (c credentialDo) Assign(attrs ...field.AssignExpr) *credentialDo {
	return c.withDO(c.DO.Assign(attrs...))
}

func (c credentialDo) Joins(fields ...field.RelationField) *credentialDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Joins(_f))
	}
	return &c
}

func (c credentialDo) Preload(fields ...field.RelationField) *credentialDo {
	for _, _f := range fields {
		c = *c.withDO(c.DO.Preload(_f))
	}
	return &c
}

func (c credentialDo) FirstOrInit() (*model.Credential, error) {
	if result, err := c.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Credential), nil
	}
}

func (c credentialDo) FirstOrCreate() (*model.Credential, error) {
	if result, err := c.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Credential), nil
	}
}

func (c credentialDo) FindByPage(offset int, limit int) (result []*model.Credential, count int64, err error) {
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

func (c credentialDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = c.Count()
	if err != nil {
		return
	}

	err = c.Offset(offset).Limit(limit).Scan(result)
	return
}

func (c credentialDo) Scan(result interface{}) (err error) {
	return c.DO.Scan(result)
}

func (c credentialDo) Delete(models ...*model.Credential) (result gen.ResultInfo, err error) {
	return c.DO.Delete(models)
}

func (c *credentialDo) withDO(do gen.Dao) *credentialDo {
	c.DO = *do.(*gen.DO)
	return c
}

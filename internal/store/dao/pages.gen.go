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

func newPage(db *gorm.DB, opts ...gen.DOOption) page {
	_page := page{}

	_page.pageDo.UseDB(db, opts...)
	_page.pageDo.UseModel(&model.Page{})

	tableName := _page.pageDo.TableName()
	_page.ALL = field.NewAsterisk(tableName)
	_page.ID = field.NewInt64(tableName, "id")
	_page.PageID = field.NewString(tableName, "page_id")
	_page.UID = field.NewString(tableName, "uid")
	_page.Topic = field.NewString(tableName, "topic")
	_page.Type = field.NewField(tableName, "type")
	_page.Schema = field.NewField(tableName, "schema")
	_page.State = field.NewField(tableName, "state")
	_page.CreatedAt = field.NewTime(tableName, "created_at")
	_page.UpdatedAt = field.NewTime(tableName, "updated_at")

	_page.fillFieldMap()

	return _page
}

type page struct {
	pageDo

	ALL       field.Asterisk
	ID        field.Int64
	PageID    field.String
	UID       field.String
	Topic     field.String
	Type      field.Field
	Schema    field.Field
	State     field.Field
	CreatedAt field.Time
	UpdatedAt field.Time

	fieldMap map[string]field.Expr
}

func (p page) Table(newTableName string) *page {
	p.pageDo.UseTable(newTableName)
	return p.updateTableName(newTableName)
}

func (p page) As(alias string) *page {
	p.pageDo.DO = *(p.pageDo.As(alias).(*gen.DO))
	return p.updateTableName(alias)
}

func (p *page) updateTableName(table string) *page {
	p.ALL = field.NewAsterisk(table)
	p.ID = field.NewInt64(table, "id")
	p.PageID = field.NewString(table, "page_id")
	p.UID = field.NewString(table, "uid")
	p.Topic = field.NewString(table, "topic")
	p.Type = field.NewField(table, "type")
	p.Schema = field.NewField(table, "schema")
	p.State = field.NewField(table, "state")
	p.CreatedAt = field.NewTime(table, "created_at")
	p.UpdatedAt = field.NewTime(table, "updated_at")

	p.fillFieldMap()

	return p
}

func (p *page) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := p.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (p *page) fillFieldMap() {
	p.fieldMap = make(map[string]field.Expr, 9)
	p.fieldMap["id"] = p.ID
	p.fieldMap["page_id"] = p.PageID
	p.fieldMap["uid"] = p.UID
	p.fieldMap["topic"] = p.Topic
	p.fieldMap["type"] = p.Type
	p.fieldMap["schema"] = p.Schema
	p.fieldMap["state"] = p.State
	p.fieldMap["created_at"] = p.CreatedAt
	p.fieldMap["updated_at"] = p.UpdatedAt
}

func (p page) clone(db *gorm.DB) page {
	p.pageDo.ReplaceConnPool(db.Statement.ConnPool)
	return p
}

func (p page) replaceDB(db *gorm.DB) page {
	p.pageDo.ReplaceDB(db)
	return p
}

type pageDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (p pageDo) GetByID(id int) (result model.Page, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM pages WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = p.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (p pageDo) TakeByUidAndTopic(uid string, topic string) (result model.Page, err error) {
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
func (p pageDo) FindByUidAndTopic(uid string, topic string) (result []*model.Page, err error) {
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

func (p pageDo) Debug() *pageDo {
	return p.withDO(p.DO.Debug())
}

func (p pageDo) WithContext(ctx context.Context) *pageDo {
	return p.withDO(p.DO.WithContext(ctx))
}

func (p pageDo) ReadDB() *pageDo {
	return p.Clauses(dbresolver.Read)
}

func (p pageDo) WriteDB() *pageDo {
	return p.Clauses(dbresolver.Write)
}

func (p pageDo) Session(config *gorm.Session) *pageDo {
	return p.withDO(p.DO.Session(config))
}

func (p pageDo) Clauses(conds ...clause.Expression) *pageDo {
	return p.withDO(p.DO.Clauses(conds...))
}

func (p pageDo) Returning(value interface{}, columns ...string) *pageDo {
	return p.withDO(p.DO.Returning(value, columns...))
}

func (p pageDo) Not(conds ...gen.Condition) *pageDo {
	return p.withDO(p.DO.Not(conds...))
}

func (p pageDo) Or(conds ...gen.Condition) *pageDo {
	return p.withDO(p.DO.Or(conds...))
}

func (p pageDo) Select(conds ...field.Expr) *pageDo {
	return p.withDO(p.DO.Select(conds...))
}

func (p pageDo) Where(conds ...gen.Condition) *pageDo {
	return p.withDO(p.DO.Where(conds...))
}

func (p pageDo) Order(conds ...field.Expr) *pageDo {
	return p.withDO(p.DO.Order(conds...))
}

func (p pageDo) Distinct(cols ...field.Expr) *pageDo {
	return p.withDO(p.DO.Distinct(cols...))
}

func (p pageDo) Omit(cols ...field.Expr) *pageDo {
	return p.withDO(p.DO.Omit(cols...))
}

func (p pageDo) Join(table schema.Tabler, on ...field.Expr) *pageDo {
	return p.withDO(p.DO.Join(table, on...))
}

func (p pageDo) LeftJoin(table schema.Tabler, on ...field.Expr) *pageDo {
	return p.withDO(p.DO.LeftJoin(table, on...))
}

func (p pageDo) RightJoin(table schema.Tabler, on ...field.Expr) *pageDo {
	return p.withDO(p.DO.RightJoin(table, on...))
}

func (p pageDo) Group(cols ...field.Expr) *pageDo {
	return p.withDO(p.DO.Group(cols...))
}

func (p pageDo) Having(conds ...gen.Condition) *pageDo {
	return p.withDO(p.DO.Having(conds...))
}

func (p pageDo) Limit(limit int) *pageDo {
	return p.withDO(p.DO.Limit(limit))
}

func (p pageDo) Offset(offset int) *pageDo {
	return p.withDO(p.DO.Offset(offset))
}

func (p pageDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *pageDo {
	return p.withDO(p.DO.Scopes(funcs...))
}

func (p pageDo) Unscoped() *pageDo {
	return p.withDO(p.DO.Unscoped())
}

func (p pageDo) Create(values ...*model.Page) error {
	if len(values) == 0 {
		return nil
	}
	return p.DO.Create(values)
}

func (p pageDo) CreateInBatches(values []*model.Page, batchSize int) error {
	return p.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (p pageDo) Save(values ...*model.Page) error {
	if len(values) == 0 {
		return nil
	}
	return p.DO.Save(values)
}

func (p pageDo) First() (*model.Page, error) {
	if result, err := p.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Page), nil
	}
}

func (p pageDo) Take() (*model.Page, error) {
	if result, err := p.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Page), nil
	}
}

func (p pageDo) Last() (*model.Page, error) {
	if result, err := p.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Page), nil
	}
}

func (p pageDo) Find() ([]*model.Page, error) {
	result, err := p.DO.Find()
	return result.([]*model.Page), err
}

func (p pageDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Page, err error) {
	buf := make([]*model.Page, 0, batchSize)
	err = p.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (p pageDo) FindInBatches(result *[]*model.Page, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return p.DO.FindInBatches(result, batchSize, fc)
}

func (p pageDo) Attrs(attrs ...field.AssignExpr) *pageDo {
	return p.withDO(p.DO.Attrs(attrs...))
}

func (p pageDo) Assign(attrs ...field.AssignExpr) *pageDo {
	return p.withDO(p.DO.Assign(attrs...))
}

func (p pageDo) Joins(fields ...field.RelationField) *pageDo {
	for _, _f := range fields {
		p = *p.withDO(p.DO.Joins(_f))
	}
	return &p
}

func (p pageDo) Preload(fields ...field.RelationField) *pageDo {
	for _, _f := range fields {
		p = *p.withDO(p.DO.Preload(_f))
	}
	return &p
}

func (p pageDo) FirstOrInit() (*model.Page, error) {
	if result, err := p.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Page), nil
	}
}

func (p pageDo) FirstOrCreate() (*model.Page, error) {
	if result, err := p.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Page), nil
	}
}

func (p pageDo) FindByPage(offset int, limit int) (result []*model.Page, count int64, err error) {
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

func (p pageDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = p.Count()
	if err != nil {
		return
	}

	err = p.Offset(offset).Limit(limit).Scan(result)
	return
}

func (p pageDo) Scan(result interface{}) (err error) {
	return p.DO.Scan(result)
}

func (p pageDo) Delete(models ...*model.Page) (result gen.ResultInfo, err error) {
	return p.DO.Delete(models)
}

func (p *pageDo) withDO(do gen.Dao) *pageDo {
	p.DO = *do.(*gen.DO)
	return p
}

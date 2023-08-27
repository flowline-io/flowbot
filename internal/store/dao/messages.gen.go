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

func newMessage(db *gorm.DB, opts ...gen.DOOption) message {
	_message := message{}

	_message.messageDo.UseDB(db, opts...)
	_message.messageDo.UseModel(&model.Message{})

	tableName := _message.messageDo.TableName()
	_message.ALL = field.NewAsterisk(tableName)
	_message.ID = field.NewInt32(tableName, "id")
	_message.Createdat = field.NewTime(tableName, "createdat")
	_message.Updatedat = field.NewTime(tableName, "updatedat")
	_message.Deletedat = field.NewTime(tableName, "deletedat")
	_message.Delid = field.NewInt32(tableName, "delid")
	_message.Seqid = field.NewInt32(tableName, "seqid")
	_message.Topic = field.NewString(tableName, "topic")
	_message.From = field.NewInt64(tableName, "from")
	_message.Head = field.NewField(tableName, "head")
	_message.Content = field.NewField(tableName, "content")

	_message.fillFieldMap()

	return _message
}

type message struct {
	messageDo

	ALL       field.Asterisk
	ID        field.Int32
	Createdat field.Time
	Updatedat field.Time
	Deletedat field.Time
	Delid     field.Int32
	Seqid     field.Int32
	Topic     field.String
	From      field.Int64
	Head      field.Field
	Content   field.Field

	fieldMap map[string]field.Expr
}

func (m message) Table(newTableName string) *message {
	m.messageDo.UseTable(newTableName)
	return m.updateTableName(newTableName)
}

func (m message) As(alias string) *message {
	m.messageDo.DO = *(m.messageDo.As(alias).(*gen.DO))
	return m.updateTableName(alias)
}

func (m *message) updateTableName(table string) *message {
	m.ALL = field.NewAsterisk(table)
	m.ID = field.NewInt32(table, "id")
	m.Createdat = field.NewTime(table, "createdat")
	m.Updatedat = field.NewTime(table, "updatedat")
	m.Deletedat = field.NewTime(table, "deletedat")
	m.Delid = field.NewInt32(table, "delid")
	m.Seqid = field.NewInt32(table, "seqid")
	m.Topic = field.NewString(table, "topic")
	m.From = field.NewInt64(table, "from")
	m.Head = field.NewField(table, "head")
	m.Content = field.NewField(table, "content")

	m.fillFieldMap()

	return m
}

func (m *message) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := m.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (m *message) fillFieldMap() {
	m.fieldMap = make(map[string]field.Expr, 12)
	m.fieldMap["id"] = m.ID
	m.fieldMap["createdat"] = m.Createdat
	m.fieldMap["updatedat"] = m.Updatedat
	m.fieldMap["deletedat"] = m.Deletedat
	m.fieldMap["delid"] = m.Delid
	m.fieldMap["seqid"] = m.Seqid
	m.fieldMap["topic"] = m.Topic
	m.fieldMap["from"] = m.From
	m.fieldMap["head"] = m.Head
	m.fieldMap["content"] = m.Content

}

func (m message) clone(db *gorm.DB) message {
	m.messageDo.ReplaceConnPool(db.Statement.ConnPool)
	return m
}

func (m message) replaceDB(db *gorm.DB) message {
	m.messageDo.ReplaceDB(db)
	return m
}

type messageDo struct{ gen.DO }

func (m messageDo) Debug() *messageDo {
	return m.withDO(m.DO.Debug())
}

func (m messageDo) WithContext(ctx context.Context) *messageDo {
	return m.withDO(m.DO.WithContext(ctx))
}

func (m messageDo) ReadDB() *messageDo {
	return m.Clauses(dbresolver.Read)
}

func (m messageDo) WriteDB() *messageDo {
	return m.Clauses(dbresolver.Write)
}

func (m messageDo) Session(config *gorm.Session) *messageDo {
	return m.withDO(m.DO.Session(config))
}

func (m messageDo) Clauses(conds ...clause.Expression) *messageDo {
	return m.withDO(m.DO.Clauses(conds...))
}

func (m messageDo) Returning(value interface{}, columns ...string) *messageDo {
	return m.withDO(m.DO.Returning(value, columns...))
}

func (m messageDo) Not(conds ...gen.Condition) *messageDo {
	return m.withDO(m.DO.Not(conds...))
}

func (m messageDo) Or(conds ...gen.Condition) *messageDo {
	return m.withDO(m.DO.Or(conds...))
}

func (m messageDo) Select(conds ...field.Expr) *messageDo {
	return m.withDO(m.DO.Select(conds...))
}

func (m messageDo) Where(conds ...gen.Condition) *messageDo {
	return m.withDO(m.DO.Where(conds...))
}

func (m messageDo) Order(conds ...field.Expr) *messageDo {
	return m.withDO(m.DO.Order(conds...))
}

func (m messageDo) Distinct(cols ...field.Expr) *messageDo {
	return m.withDO(m.DO.Distinct(cols...))
}

func (m messageDo) Omit(cols ...field.Expr) *messageDo {
	return m.withDO(m.DO.Omit(cols...))
}

func (m messageDo) Join(table schema.Tabler, on ...field.Expr) *messageDo {
	return m.withDO(m.DO.Join(table, on...))
}

func (m messageDo) LeftJoin(table schema.Tabler, on ...field.Expr) *messageDo {
	return m.withDO(m.DO.LeftJoin(table, on...))
}

func (m messageDo) RightJoin(table schema.Tabler, on ...field.Expr) *messageDo {
	return m.withDO(m.DO.RightJoin(table, on...))
}

func (m messageDo) Group(cols ...field.Expr) *messageDo {
	return m.withDO(m.DO.Group(cols...))
}

func (m messageDo) Having(conds ...gen.Condition) *messageDo {
	return m.withDO(m.DO.Having(conds...))
}

func (m messageDo) Limit(limit int) *messageDo {
	return m.withDO(m.DO.Limit(limit))
}

func (m messageDo) Offset(offset int) *messageDo {
	return m.withDO(m.DO.Offset(offset))
}

func (m messageDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *messageDo {
	return m.withDO(m.DO.Scopes(funcs...))
}

func (m messageDo) Unscoped() *messageDo {
	return m.withDO(m.DO.Unscoped())
}

func (m messageDo) Create(values ...*model.Message) error {
	if len(values) == 0 {
		return nil
	}
	return m.DO.Create(values)
}

func (m messageDo) CreateInBatches(values []*model.Message, batchSize int) error {
	return m.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (m messageDo) Save(values ...*model.Message) error {
	if len(values) == 0 {
		return nil
	}
	return m.DO.Save(values)
}

func (m messageDo) First() (*model.Message, error) {
	if result, err := m.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Message), nil
	}
}

func (m messageDo) Take() (*model.Message, error) {
	if result, err := m.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Message), nil
	}
}

func (m messageDo) Last() (*model.Message, error) {
	if result, err := m.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Message), nil
	}
}

func (m messageDo) Find() ([]*model.Message, error) {
	result, err := m.DO.Find()
	return result.([]*model.Message), err
}

func (m messageDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Message, err error) {
	buf := make([]*model.Message, 0, batchSize)
	err = m.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (m messageDo) FindInBatches(result *[]*model.Message, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return m.DO.FindInBatches(result, batchSize, fc)
}

func (m messageDo) Attrs(attrs ...field.AssignExpr) *messageDo {
	return m.withDO(m.DO.Attrs(attrs...))
}

func (m messageDo) Assign(attrs ...field.AssignExpr) *messageDo {
	return m.withDO(m.DO.Assign(attrs...))
}

func (m messageDo) Joins(fields ...field.RelationField) *messageDo {
	for _, _f := range fields {
		m = *m.withDO(m.DO.Joins(_f))
	}
	return &m
}

func (m messageDo) Preload(fields ...field.RelationField) *messageDo {
	for _, _f := range fields {
		m = *m.withDO(m.DO.Preload(_f))
	}
	return &m
}

func (m messageDo) FirstOrInit() (*model.Message, error) {
	if result, err := m.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Message), nil
	}
}

func (m messageDo) FirstOrCreate() (*model.Message, error) {
	if result, err := m.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Message), nil
	}
}

func (m messageDo) FindByPage(offset int, limit int) (result []*model.Message, count int64, err error) {
	result, err = m.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = m.Offset(-1).Limit(-1).Count()
	return
}

func (m messageDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = m.Count()
	if err != nil {
		return
	}

	err = m.Offset(offset).Limit(limit).Scan(result)
	return
}

func (m messageDo) Scan(result interface{}) (err error) {
	return m.DO.Scan(result)
}

func (m messageDo) Delete(models ...*model.Message) (result gen.ResultInfo, err error) {
	return m.DO.Delete(models)
}

func (m *messageDo) withDO(do gen.Dao) *messageDo {
	m.DO = *do.(*gen.DO)
	return m
}

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

func newWorkflow(db *gorm.DB, opts ...gen.DOOption) workflow {
	_workflow := workflow{}

	_workflow.workflowDo.UseDB(db, opts...)
	_workflow.workflowDo.UseModel(&model.Workflow{})

	tableName := _workflow.workflowDo.TableName()
	_workflow.ALL = field.NewAsterisk(tableName)
	_workflow.ID = field.NewInt64(tableName, "id")
	_workflow.UID = field.NewString(tableName, "uid")
	_workflow.Topic = field.NewString(tableName, "topic")
	_workflow.Flag = field.NewString(tableName, "flag")
	_workflow.Name = field.NewString(tableName, "name")
	_workflow.Describe = field.NewString(tableName, "describe")
	_workflow.SuccessfulCount = field.NewInt32(tableName, "successful_count")
	_workflow.FailedCount = field.NewInt32(tableName, "failed_count")
	_workflow.RunningCount = field.NewInt32(tableName, "running_count")
	_workflow.CanceledCount = field.NewInt32(tableName, "canceled_count")
	_workflow.State = field.NewField(tableName, "state")
	_workflow.CreatedAt = field.NewTime(tableName, "created_at")
	_workflow.UpdatedAt = field.NewTime(tableName, "updated_at")
	_workflow.Dag = workflowHasOneDag{
		db: db.Session(&gorm.Session{}),

		RelationField: field.NewRelation("Dag", "model.Dag"),
	}

	_workflow.Triggers = workflowHasManyTriggers{
		db: db.Session(&gorm.Session{}),

		RelationField: field.NewRelation("Triggers", "model.WorkflowTrigger"),
	}

	_workflow.fillFieldMap()

	return _workflow
}

type workflow struct {
	workflowDo

	ALL             field.Asterisk
	ID              field.Int64
	UID             field.String
	Topic           field.String
	Flag            field.String
	Name            field.String
	Describe        field.String
	SuccessfulCount field.Int32
	FailedCount     field.Int32
	RunningCount    field.Int32
	CanceledCount   field.Int32
	State           field.Field
	CreatedAt       field.Time
	UpdatedAt       field.Time
	Dag             workflowHasOneDag

	Triggers workflowHasManyTriggers

	fieldMap map[string]field.Expr
}

func (w workflow) Table(newTableName string) *workflow {
	w.workflowDo.UseTable(newTableName)
	return w.updateTableName(newTableName)
}

func (w workflow) As(alias string) *workflow {
	w.workflowDo.DO = *(w.workflowDo.As(alias).(*gen.DO))
	return w.updateTableName(alias)
}

func (w *workflow) updateTableName(table string) *workflow {
	w.ALL = field.NewAsterisk(table)
	w.ID = field.NewInt64(table, "id")
	w.UID = field.NewString(table, "uid")
	w.Topic = field.NewString(table, "topic")
	w.Flag = field.NewString(table, "flag")
	w.Name = field.NewString(table, "name")
	w.Describe = field.NewString(table, "describe")
	w.SuccessfulCount = field.NewInt32(table, "successful_count")
	w.FailedCount = field.NewInt32(table, "failed_count")
	w.RunningCount = field.NewInt32(table, "running_count")
	w.CanceledCount = field.NewInt32(table, "canceled_count")
	w.State = field.NewField(table, "state")
	w.CreatedAt = field.NewTime(table, "created_at")
	w.UpdatedAt = field.NewTime(table, "updated_at")

	w.fillFieldMap()

	return w
}

func (w *workflow) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := w.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (w *workflow) fillFieldMap() {
	w.fieldMap = make(map[string]field.Expr, 15)
	w.fieldMap["id"] = w.ID
	w.fieldMap["uid"] = w.UID
	w.fieldMap["topic"] = w.Topic
	w.fieldMap["flag"] = w.Flag
	w.fieldMap["name"] = w.Name
	w.fieldMap["describe"] = w.Describe
	w.fieldMap["successful_count"] = w.SuccessfulCount
	w.fieldMap["failed_count"] = w.FailedCount
	w.fieldMap["running_count"] = w.RunningCount
	w.fieldMap["canceled_count"] = w.CanceledCount
	w.fieldMap["state"] = w.State
	w.fieldMap["created_at"] = w.CreatedAt
	w.fieldMap["updated_at"] = w.UpdatedAt

}

func (w workflow) clone(db *gorm.DB) workflow {
	w.workflowDo.ReplaceConnPool(db.Statement.ConnPool)
	w.Dag.db = db.Session(&gorm.Session{Initialized: true})
	w.Dag.db.Statement.ConnPool = db.Statement.ConnPool
	w.Triggers.db = db.Session(&gorm.Session{Initialized: true})
	w.Triggers.db.Statement.ConnPool = db.Statement.ConnPool
	return w
}

func (w workflow) replaceDB(db *gorm.DB) workflow {
	w.workflowDo.ReplaceDB(db)
	w.Dag.db = db.Session(&gorm.Session{})
	w.Triggers.db = db.Session(&gorm.Session{})
	return w
}

type workflowHasOneDag struct {
	db *gorm.DB

	field.RelationField
}

func (a workflowHasOneDag) Where(conds ...field.Expr) *workflowHasOneDag {
	if len(conds) == 0 {
		return &a
	}

	exprs := make([]clause.Expression, 0, len(conds))
	for _, cond := range conds {
		exprs = append(exprs, cond.BeCond().(clause.Expression))
	}
	a.db = a.db.Clauses(clause.Where{Exprs: exprs})
	return &a
}

func (a workflowHasOneDag) WithContext(ctx context.Context) *workflowHasOneDag {
	a.db = a.db.WithContext(ctx)
	return &a
}

func (a workflowHasOneDag) Session(session *gorm.Session) *workflowHasOneDag {
	a.db = a.db.Session(session)
	return &a
}

func (a workflowHasOneDag) Model(m *model.Workflow) *workflowHasOneDagTx {
	return &workflowHasOneDagTx{a.db.Model(m).Association(a.Name())}
}

func (a workflowHasOneDag) Unscoped() *workflowHasOneDag {
	a.db = a.db.Unscoped()
	return &a
}

type workflowHasOneDagTx struct{ tx *gorm.Association }

func (a workflowHasOneDagTx) Find() (result *model.Dag, err error) {
	return result, a.tx.Find(&result)
}

func (a workflowHasOneDagTx) Append(values ...*model.Dag) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Append(targetValues...)
}

func (a workflowHasOneDagTx) Replace(values ...*model.Dag) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Replace(targetValues...)
}

func (a workflowHasOneDagTx) Delete(values ...*model.Dag) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Delete(targetValues...)
}

func (a workflowHasOneDagTx) Clear() error {
	return a.tx.Clear()
}

func (a workflowHasOneDagTx) Count() int64 {
	return a.tx.Count()
}

func (a workflowHasOneDagTx) Unscoped() *workflowHasOneDagTx {
	a.tx = a.tx.Unscoped()
	return &a
}

type workflowHasManyTriggers struct {
	db *gorm.DB

	field.RelationField
}

func (a workflowHasManyTriggers) Where(conds ...field.Expr) *workflowHasManyTriggers {
	if len(conds) == 0 {
		return &a
	}

	exprs := make([]clause.Expression, 0, len(conds))
	for _, cond := range conds {
		exprs = append(exprs, cond.BeCond().(clause.Expression))
	}
	a.db = a.db.Clauses(clause.Where{Exprs: exprs})
	return &a
}

func (a workflowHasManyTriggers) WithContext(ctx context.Context) *workflowHasManyTriggers {
	a.db = a.db.WithContext(ctx)
	return &a
}

func (a workflowHasManyTriggers) Session(session *gorm.Session) *workflowHasManyTriggers {
	a.db = a.db.Session(session)
	return &a
}

func (a workflowHasManyTriggers) Model(m *model.Workflow) *workflowHasManyTriggersTx {
	return &workflowHasManyTriggersTx{a.db.Model(m).Association(a.Name())}
}

func (a workflowHasManyTriggers) Unscoped() *workflowHasManyTriggers {
	a.db = a.db.Unscoped()
	return &a
}

type workflowHasManyTriggersTx struct{ tx *gorm.Association }

func (a workflowHasManyTriggersTx) Find() (result []*model.WorkflowTrigger, err error) {
	return result, a.tx.Find(&result)
}

func (a workflowHasManyTriggersTx) Append(values ...*model.WorkflowTrigger) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Append(targetValues...)
}

func (a workflowHasManyTriggersTx) Replace(values ...*model.WorkflowTrigger) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Replace(targetValues...)
}

func (a workflowHasManyTriggersTx) Delete(values ...*model.WorkflowTrigger) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Delete(targetValues...)
}

func (a workflowHasManyTriggersTx) Clear() error {
	return a.tx.Clear()
}

func (a workflowHasManyTriggersTx) Count() int64 {
	return a.tx.Count()
}

func (a workflowHasManyTriggersTx) Unscoped() *workflowHasManyTriggersTx {
	a.tx = a.tx.Unscoped()
	return &a
}

type workflowDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (w workflowDo) GetByID(id int) (result model.Workflow, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM workflow WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = w.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (w workflowDo) TakeByUidAndTopic(uid string, topic string) (result model.Workflow, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = w.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (w workflowDo) FindByUidAndTopic(uid string, topic string) (result []*model.Workflow, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = w.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (w workflowDo) Debug() *workflowDo {
	return w.withDO(w.DO.Debug())
}

func (w workflowDo) WithContext(ctx context.Context) *workflowDo {
	return w.withDO(w.DO.WithContext(ctx))
}

func (w workflowDo) ReadDB() *workflowDo {
	return w.Clauses(dbresolver.Read)
}

func (w workflowDo) WriteDB() *workflowDo {
	return w.Clauses(dbresolver.Write)
}

func (w workflowDo) Session(config *gorm.Session) *workflowDo {
	return w.withDO(w.DO.Session(config))
}

func (w workflowDo) Clauses(conds ...clause.Expression) *workflowDo {
	return w.withDO(w.DO.Clauses(conds...))
}

func (w workflowDo) Returning(value interface{}, columns ...string) *workflowDo {
	return w.withDO(w.DO.Returning(value, columns...))
}

func (w workflowDo) Not(conds ...gen.Condition) *workflowDo {
	return w.withDO(w.DO.Not(conds...))
}

func (w workflowDo) Or(conds ...gen.Condition) *workflowDo {
	return w.withDO(w.DO.Or(conds...))
}

func (w workflowDo) Select(conds ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Select(conds...))
}

func (w workflowDo) Where(conds ...gen.Condition) *workflowDo {
	return w.withDO(w.DO.Where(conds...))
}

func (w workflowDo) Order(conds ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Order(conds...))
}

func (w workflowDo) Distinct(cols ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Distinct(cols...))
}

func (w workflowDo) Omit(cols ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Omit(cols...))
}

func (w workflowDo) Join(table schema.Tabler, on ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Join(table, on...))
}

func (w workflowDo) LeftJoin(table schema.Tabler, on ...field.Expr) *workflowDo {
	return w.withDO(w.DO.LeftJoin(table, on...))
}

func (w workflowDo) RightJoin(table schema.Tabler, on ...field.Expr) *workflowDo {
	return w.withDO(w.DO.RightJoin(table, on...))
}

func (w workflowDo) Group(cols ...field.Expr) *workflowDo {
	return w.withDO(w.DO.Group(cols...))
}

func (w workflowDo) Having(conds ...gen.Condition) *workflowDo {
	return w.withDO(w.DO.Having(conds...))
}

func (w workflowDo) Limit(limit int) *workflowDo {
	return w.withDO(w.DO.Limit(limit))
}

func (w workflowDo) Offset(offset int) *workflowDo {
	return w.withDO(w.DO.Offset(offset))
}

func (w workflowDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *workflowDo {
	return w.withDO(w.DO.Scopes(funcs...))
}

func (w workflowDo) Unscoped() *workflowDo {
	return w.withDO(w.DO.Unscoped())
}

func (w workflowDo) Create(values ...*model.Workflow) error {
	if len(values) == 0 {
		return nil
	}
	return w.DO.Create(values)
}

func (w workflowDo) CreateInBatches(values []*model.Workflow, batchSize int) error {
	return w.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (w workflowDo) Save(values ...*model.Workflow) error {
	if len(values) == 0 {
		return nil
	}
	return w.DO.Save(values)
}

func (w workflowDo) First() (*model.Workflow, error) {
	if result, err := w.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Workflow), nil
	}
}

func (w workflowDo) Take() (*model.Workflow, error) {
	if result, err := w.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Workflow), nil
	}
}

func (w workflowDo) Last() (*model.Workflow, error) {
	if result, err := w.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Workflow), nil
	}
}

func (w workflowDo) Find() ([]*model.Workflow, error) {
	result, err := w.DO.Find()
	return result.([]*model.Workflow), err
}

func (w workflowDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Workflow, err error) {
	buf := make([]*model.Workflow, 0, batchSize)
	err = w.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (w workflowDo) FindInBatches(result *[]*model.Workflow, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return w.DO.FindInBatches(result, batchSize, fc)
}

func (w workflowDo) Attrs(attrs ...field.AssignExpr) *workflowDo {
	return w.withDO(w.DO.Attrs(attrs...))
}

func (w workflowDo) Assign(attrs ...field.AssignExpr) *workflowDo {
	return w.withDO(w.DO.Assign(attrs...))
}

func (w workflowDo) Joins(fields ...field.RelationField) *workflowDo {
	for _, _f := range fields {
		w = *w.withDO(w.DO.Joins(_f))
	}
	return &w
}

func (w workflowDo) Preload(fields ...field.RelationField) *workflowDo {
	for _, _f := range fields {
		w = *w.withDO(w.DO.Preload(_f))
	}
	return &w
}

func (w workflowDo) FirstOrInit() (*model.Workflow, error) {
	if result, err := w.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Workflow), nil
	}
}

func (w workflowDo) FirstOrCreate() (*model.Workflow, error) {
	if result, err := w.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Workflow), nil
	}
}

func (w workflowDo) FindByPage(offset int, limit int) (result []*model.Workflow, count int64, err error) {
	result, err = w.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = w.Offset(-1).Limit(-1).Count()
	return
}

func (w workflowDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = w.Count()
	if err != nil {
		return
	}

	err = w.Offset(offset).Limit(limit).Scan(result)
	return
}

func (w workflowDo) Scan(result interface{}) (err error) {
	return w.DO.Scan(result)
}

func (w workflowDo) Delete(models ...*model.Workflow) (result gen.ResultInfo, err error) {
	return w.DO.Delete(models)
}

func (w *workflowDo) withDO(do gen.Dao) *workflowDo {
	w.DO = *do.(*gen.DO)
	return w
}

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

func newObjective(db *gorm.DB, opts ...gen.DOOption) objective {
	_objective := objective{}

	_objective.objectiveDo.UseDB(db, opts...)
	_objective.objectiveDo.UseModel(&model.Objective{})

	tableName := _objective.objectiveDo.TableName()
	_objective.ALL = field.NewAsterisk(tableName)
	_objective.ID = field.NewInt64(tableName, "id")
	_objective.UID = field.NewString(tableName, "uid")
	_objective.Topic = field.NewString(tableName, "topic")
	_objective.Sequence = field.NewInt32(tableName, "sequence")
	_objective.Progress = field.NewInt32(tableName, "progress")
	_objective.Title = field.NewString(tableName, "title")
	_objective.Memo = field.NewString(tableName, "memo")
	_objective.Motive = field.NewString(tableName, "motive")
	_objective.Feasibility = field.NewString(tableName, "feasibility")
	_objective.IsPlan = field.NewInt32(tableName, "is_plan")
	_objective.PlanStart = field.NewTime(tableName, "plan_start")
	_objective.PlanEnd = field.NewTime(tableName, "plan_end")
	_objective.TotalValue = field.NewInt32(tableName, "total_value")
	_objective.CurrentValue = field.NewInt32(tableName, "current_value")
	_objective.Tag = field.NewString(tableName, "tag")
	_objective.CreatedData = field.NewTime(tableName, "created_data")
	_objective.UpdatedDate = field.NewTime(tableName, "updated_date")
	_objective.KeyResults = objectiveHasManyKeyResults{
		db: db.Session(&gorm.Session{}),

		RelationField: field.NewRelation("KeyResults", "model.KeyResult"),
		KeyResultValues: struct {
			field.RelationField
		}{
			RelationField: field.NewRelation("KeyResults.KeyResultValues", "model.KeyResultValue"),
		},
		Todos: struct {
			field.RelationField
			SubTodos struct {
				field.RelationField
			}
		}{
			RelationField: field.NewRelation("KeyResults.Todos", "model.Todo"),
			SubTodos: struct {
				field.RelationField
			}{
				RelationField: field.NewRelation("KeyResults.Todos.SubTodos", "model.Todo"),
			},
		},
	}

	_objective.Reviews = objectiveHasManyReviews{
		db: db.Session(&gorm.Session{}),

		RelationField: field.NewRelation("Reviews", "model.Review"),
		Evaluations: struct {
			field.RelationField
		}{
			RelationField: field.NewRelation("Reviews.Evaluations", "model.ReviewEvaluation"),
		},
	}

	_objective.fillFieldMap()

	return _objective
}

type objective struct {
	objectiveDo

	ALL          field.Asterisk
	ID           field.Int64
	UID          field.String
	Topic        field.String
	Sequence     field.Int32
	Progress     field.Int32
	Title        field.String
	Memo         field.String
	Motive       field.String
	Feasibility  field.String
	IsPlan       field.Int32
	PlanStart    field.Time
	PlanEnd      field.Time
	TotalValue   field.Int32
	CurrentValue field.Int32
	Tag          field.String
	CreatedData  field.Time
	UpdatedDate  field.Time
	KeyResults   objectiveHasManyKeyResults

	Reviews objectiveHasManyReviews

	fieldMap map[string]field.Expr
}

func (o objective) Table(newTableName string) *objective {
	o.objectiveDo.UseTable(newTableName)
	return o.updateTableName(newTableName)
}

func (o objective) As(alias string) *objective {
	o.objectiveDo.DO = *(o.objectiveDo.As(alias).(*gen.DO))
	return o.updateTableName(alias)
}

func (o *objective) updateTableName(table string) *objective {
	o.ALL = field.NewAsterisk(table)
	o.ID = field.NewInt64(table, "id")
	o.UID = field.NewString(table, "uid")
	o.Topic = field.NewString(table, "topic")
	o.Sequence = field.NewInt32(table, "sequence")
	o.Progress = field.NewInt32(table, "progress")
	o.Title = field.NewString(table, "title")
	o.Memo = field.NewString(table, "memo")
	o.Motive = field.NewString(table, "motive")
	o.Feasibility = field.NewString(table, "feasibility")
	o.IsPlan = field.NewInt32(table, "is_plan")
	o.PlanStart = field.NewTime(table, "plan_start")
	o.PlanEnd = field.NewTime(table, "plan_end")
	o.TotalValue = field.NewInt32(table, "total_value")
	o.CurrentValue = field.NewInt32(table, "current_value")
	o.Tag = field.NewString(table, "tag")
	o.CreatedData = field.NewTime(table, "created_data")
	o.UpdatedDate = field.NewTime(table, "updated_date")

	o.fillFieldMap()

	return o
}

func (o *objective) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := o.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (o *objective) fillFieldMap() {
	o.fieldMap = make(map[string]field.Expr, 19)
	o.fieldMap["id"] = o.ID
	o.fieldMap["uid"] = o.UID
	o.fieldMap["topic"] = o.Topic
	o.fieldMap["sequence"] = o.Sequence
	o.fieldMap["progress"] = o.Progress
	o.fieldMap["title"] = o.Title
	o.fieldMap["memo"] = o.Memo
	o.fieldMap["motive"] = o.Motive
	o.fieldMap["feasibility"] = o.Feasibility
	o.fieldMap["is_plan"] = o.IsPlan
	o.fieldMap["plan_start"] = o.PlanStart
	o.fieldMap["plan_end"] = o.PlanEnd
	o.fieldMap["total_value"] = o.TotalValue
	o.fieldMap["current_value"] = o.CurrentValue
	o.fieldMap["tag"] = o.Tag
	o.fieldMap["created_data"] = o.CreatedData
	o.fieldMap["updated_date"] = o.UpdatedDate

}

func (o objective) clone(db *gorm.DB) objective {
	o.objectiveDo.ReplaceConnPool(db.Statement.ConnPool)
	return o
}

func (o objective) replaceDB(db *gorm.DB) objective {
	o.objectiveDo.ReplaceDB(db)
	return o
}

type objectiveHasManyKeyResults struct {
	db *gorm.DB

	field.RelationField

	KeyResultValues struct {
		field.RelationField
	}
	Todos struct {
		field.RelationField
		SubTodos struct {
			field.RelationField
		}
	}
}

func (a objectiveHasManyKeyResults) Where(conds ...field.Expr) *objectiveHasManyKeyResults {
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

func (a objectiveHasManyKeyResults) WithContext(ctx context.Context) *objectiveHasManyKeyResults {
	a.db = a.db.WithContext(ctx)
	return &a
}

func (a objectiveHasManyKeyResults) Session(session *gorm.Session) *objectiveHasManyKeyResults {
	a.db = a.db.Session(session)
	return &a
}

func (a objectiveHasManyKeyResults) Model(m *model.Objective) *objectiveHasManyKeyResultsTx {
	return &objectiveHasManyKeyResultsTx{a.db.Model(m).Association(a.Name())}
}

type objectiveHasManyKeyResultsTx struct{ tx *gorm.Association }

func (a objectiveHasManyKeyResultsTx) Find() (result []*model.KeyResult, err error) {
	return result, a.tx.Find(&result)
}

func (a objectiveHasManyKeyResultsTx) Append(values ...*model.KeyResult) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Append(targetValues...)
}

func (a objectiveHasManyKeyResultsTx) Replace(values ...*model.KeyResult) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Replace(targetValues...)
}

func (a objectiveHasManyKeyResultsTx) Delete(values ...*model.KeyResult) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Delete(targetValues...)
}

func (a objectiveHasManyKeyResultsTx) Clear() error {
	return a.tx.Clear()
}

func (a objectiveHasManyKeyResultsTx) Count() int64 {
	return a.tx.Count()
}

type objectiveHasManyReviews struct {
	db *gorm.DB

	field.RelationField

	Evaluations struct {
		field.RelationField
	}
}

func (a objectiveHasManyReviews) Where(conds ...field.Expr) *objectiveHasManyReviews {
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

func (a objectiveHasManyReviews) WithContext(ctx context.Context) *objectiveHasManyReviews {
	a.db = a.db.WithContext(ctx)
	return &a
}

func (a objectiveHasManyReviews) Session(session *gorm.Session) *objectiveHasManyReviews {
	a.db = a.db.Session(session)
	return &a
}

func (a objectiveHasManyReviews) Model(m *model.Objective) *objectiveHasManyReviewsTx {
	return &objectiveHasManyReviewsTx{a.db.Model(m).Association(a.Name())}
}

type objectiveHasManyReviewsTx struct{ tx *gorm.Association }

func (a objectiveHasManyReviewsTx) Find() (result []*model.Review, err error) {
	return result, a.tx.Find(&result)
}

func (a objectiveHasManyReviewsTx) Append(values ...*model.Review) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Append(targetValues...)
}

func (a objectiveHasManyReviewsTx) Replace(values ...*model.Review) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Replace(targetValues...)
}

func (a objectiveHasManyReviewsTx) Delete(values ...*model.Review) (err error) {
	targetValues := make([]interface{}, len(values))
	for i, v := range values {
		targetValues[i] = v
	}
	return a.tx.Delete(targetValues...)
}

func (a objectiveHasManyReviewsTx) Clear() error {
	return a.tx.Clear()
}

func (a objectiveHasManyReviewsTx) Count() int64 {
	return a.tx.Count()
}

type objectiveDo struct{ gen.DO }

// GetByID
//
// SELECT * FROM @@table WHERE id=@id
func (o objectiveDo) GetByID(id int) (result model.Objective, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, id)
	generateSQL.WriteString("SELECT * FROM objectives WHERE id=? ")

	var executeSQL *gorm.DB
	executeSQL = o.UnderlyingDB().Raw(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// TakeByUidAndTopic query data by uid and topic and return one
//
// where("uid=@uid AND topic=@topic")
func (o objectiveDo) TakeByUidAndTopic(uid string, topic string) (result model.Objective, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = o.UnderlyingDB().Where(generateSQL.String(), params...).Take(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

// FindByUidAndTopic query data by uid and topic and return array
//
// where("uid=@uid AND topic=@topic")
func (o objectiveDo) FindByUidAndTopic(uid string, topic string) (result []*model.Objective, err error) {
	var params []interface{}

	var generateSQL strings.Builder
	params = append(params, uid)
	params = append(params, topic)
	generateSQL.WriteString("uid=? AND topic=? ")

	var executeSQL *gorm.DB
	executeSQL = o.UnderlyingDB().Where(generateSQL.String(), params...).Find(&result) // ignore_security_alert
	err = executeSQL.Error

	return
}

func (o objectiveDo) Debug() *objectiveDo {
	return o.withDO(o.DO.Debug())
}

func (o objectiveDo) WithContext(ctx context.Context) *objectiveDo {
	return o.withDO(o.DO.WithContext(ctx))
}

func (o objectiveDo) ReadDB() *objectiveDo {
	return o.Clauses(dbresolver.Read)
}

func (o objectiveDo) WriteDB() *objectiveDo {
	return o.Clauses(dbresolver.Write)
}

func (o objectiveDo) Session(config *gorm.Session) *objectiveDo {
	return o.withDO(o.DO.Session(config))
}

func (o objectiveDo) Clauses(conds ...clause.Expression) *objectiveDo {
	return o.withDO(o.DO.Clauses(conds...))
}

func (o objectiveDo) Returning(value interface{}, columns ...string) *objectiveDo {
	return o.withDO(o.DO.Returning(value, columns...))
}

func (o objectiveDo) Not(conds ...gen.Condition) *objectiveDo {
	return o.withDO(o.DO.Not(conds...))
}

func (o objectiveDo) Or(conds ...gen.Condition) *objectiveDo {
	return o.withDO(o.DO.Or(conds...))
}

func (o objectiveDo) Select(conds ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Select(conds...))
}

func (o objectiveDo) Where(conds ...gen.Condition) *objectiveDo {
	return o.withDO(o.DO.Where(conds...))
}

func (o objectiveDo) Order(conds ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Order(conds...))
}

func (o objectiveDo) Distinct(cols ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Distinct(cols...))
}

func (o objectiveDo) Omit(cols ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Omit(cols...))
}

func (o objectiveDo) Join(table schema.Tabler, on ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Join(table, on...))
}

func (o objectiveDo) LeftJoin(table schema.Tabler, on ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.LeftJoin(table, on...))
}

func (o objectiveDo) RightJoin(table schema.Tabler, on ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.RightJoin(table, on...))
}

func (o objectiveDo) Group(cols ...field.Expr) *objectiveDo {
	return o.withDO(o.DO.Group(cols...))
}

func (o objectiveDo) Having(conds ...gen.Condition) *objectiveDo {
	return o.withDO(o.DO.Having(conds...))
}

func (o objectiveDo) Limit(limit int) *objectiveDo {
	return o.withDO(o.DO.Limit(limit))
}

func (o objectiveDo) Offset(offset int) *objectiveDo {
	return o.withDO(o.DO.Offset(offset))
}

func (o objectiveDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *objectiveDo {
	return o.withDO(o.DO.Scopes(funcs...))
}

func (o objectiveDo) Unscoped() *objectiveDo {
	return o.withDO(o.DO.Unscoped())
}

func (o objectiveDo) Create(values ...*model.Objective) error {
	if len(values) == 0 {
		return nil
	}
	return o.DO.Create(values)
}

func (o objectiveDo) CreateInBatches(values []*model.Objective, batchSize int) error {
	return o.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (o objectiveDo) Save(values ...*model.Objective) error {
	if len(values) == 0 {
		return nil
	}
	return o.DO.Save(values)
}

func (o objectiveDo) First() (*model.Objective, error) {
	if result, err := o.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.Objective), nil
	}
}

func (o objectiveDo) Take() (*model.Objective, error) {
	if result, err := o.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.Objective), nil
	}
}

func (o objectiveDo) Last() (*model.Objective, error) {
	if result, err := o.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.Objective), nil
	}
}

func (o objectiveDo) Find() ([]*model.Objective, error) {
	result, err := o.DO.Find()
	return result.([]*model.Objective), err
}

func (o objectiveDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.Objective, err error) {
	buf := make([]*model.Objective, 0, batchSize)
	err = o.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (o objectiveDo) FindInBatches(result *[]*model.Objective, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return o.DO.FindInBatches(result, batchSize, fc)
}

func (o objectiveDo) Attrs(attrs ...field.AssignExpr) *objectiveDo {
	return o.withDO(o.DO.Attrs(attrs...))
}

func (o objectiveDo) Assign(attrs ...field.AssignExpr) *objectiveDo {
	return o.withDO(o.DO.Assign(attrs...))
}

func (o objectiveDo) Joins(fields ...field.RelationField) *objectiveDo {
	for _, _f := range fields {
		o = *o.withDO(o.DO.Joins(_f))
	}
	return &o
}

func (o objectiveDo) Preload(fields ...field.RelationField) *objectiveDo {
	for _, _f := range fields {
		o = *o.withDO(o.DO.Preload(_f))
	}
	return &o
}

func (o objectiveDo) FirstOrInit() (*model.Objective, error) {
	if result, err := o.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.Objective), nil
	}
}

func (o objectiveDo) FirstOrCreate() (*model.Objective, error) {
	if result, err := o.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.Objective), nil
	}
}

func (o objectiveDo) FindByPage(offset int, limit int) (result []*model.Objective, count int64, err error) {
	result, err = o.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = o.Offset(-1).Limit(-1).Count()
	return
}

func (o objectiveDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = o.Count()
	if err != nil {
		return
	}

	err = o.Offset(offset).Limit(limit).Scan(result)
	return
}

func (o objectiveDo) Scan(result interface{}) (err error) {
	return o.DO.Scan(result)
}

func (o objectiveDo) Delete(models ...*model.Objective) (result gen.ResultInfo, err error) {
	return o.DO.Delete(models)
}

func (o *objectiveDo) withDO(do gen.Dao) *objectiveDo {
	o.DO = *do.(*gen.DO)
	return o
}

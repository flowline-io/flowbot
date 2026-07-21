package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkflowTask stores one task belonging to a workflow definition.
type WorkflowTask struct {
	ent.Schema
}

// Fields of the WorkflowTask.
func (WorkflowTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id"),
		field.String("task_id").NotEmpty(),
		field.String("action").NotEmpty(),
		field.String("describe").Optional().Default(""),
		field.JSON("params", map[string]any{}).Optional(),
		field.JSON("vars", []string{}).Optional(),
		field.JSON("conn", []string{}).Optional(),
		field.JSON("retry", map[string]any{}).Optional(),
	}
}

// Indexes of the WorkflowTask.
func (WorkflowTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_id", "task_id").Unique(),
		index.Fields("workflow_id"),
	}
}

// Annotations of the WorkflowTask.
func (WorkflowTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_tasks"),
	}
}

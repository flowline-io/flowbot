package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WorkflowStepRun struct {
	ent.Schema
}

func (WorkflowStepRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_run_id"),
		field.String("step_id").NotEmpty(),
		field.String("step_name").Default(""),
		field.String("action").NotEmpty(),
		field.String("action_type").NotEmpty(),
		field.JSON("params", map[string]any{}).Optional(),
		field.JSON("result", map[string]any{}).Optional(),
		field.Int("attempt").Default(1),
		field.Int("status").Default(0),
		field.String("error").Optional().Default(""),
		field.Time("started_at"),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (WorkflowStepRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_run_id"),
	}
}

func (WorkflowStepRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_step_runs"),
	}
}

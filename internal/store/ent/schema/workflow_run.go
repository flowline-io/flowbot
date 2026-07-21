package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WorkflowRun struct {
	ent.Schema
}

func (WorkflowRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id").Optional().Nillable(),
		field.String("workflow_name").NotEmpty(),
		// workflow_file is "" or "db" for DB-backed definitions; legacy file paths may remain.
		field.String("workflow_file").Default(""),
		field.Int("status").Default(0),
		field.String("trigger_type").Default(""),
		field.JSON("trigger_info", map[string]any{}).Optional(),
		field.JSON("input_params", map[string]any{}).Optional(),
		field.JSON("checkpoint_data", map[string]any{}).Optional(),
		field.Time("last_heartbeat").Optional().Nillable(),
		field.String("error").Optional().Default(""),
		field.Time("started_at"),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (WorkflowRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_name"),
		index.Fields("workflow_id"),
	}
}

func (WorkflowRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_runs"),
	}
}

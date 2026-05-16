package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PipelineStepRun struct {
	ent.Schema
}

func (PipelineStepRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("pipeline_run_id"),
		field.String("step_name").NotEmpty(),
		field.String("capability").Default(""),
		field.String("operation").Default(""),
		field.JSON("params", map[string]any{}).Optional(),
		field.JSON("result", map[string]any{}).Optional(),
		field.Int("attempt").Default(1),
		field.JSON("retry_config", map[string]any{}).Optional(),
		field.Int("status").Default(0),
		field.String("error").Optional().Default(""),
		field.Time("started_at"),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Immutable(),
	}
}

func (PipelineStepRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("pipeline_run_id"),
	}
}

func (PipelineStepRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_step_runs"),
	}
}

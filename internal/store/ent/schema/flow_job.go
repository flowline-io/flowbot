package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type FlowJob struct {
	ent.Schema
}

func (FlowJob) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("flow_id"),
		field.String("execution_id").NotEmpty(),
		field.String("node_id").NotEmpty(),
		field.String("node_type").NotEmpty(),
		field.String("bot").NotEmpty(),
		field.String("rule_id").NotEmpty(),
		field.Int("attempt").Default(1),
		field.Int("state").Default(0),
		field.JSON("params", map[string]any{}).Optional(),
		field.JSON("result", map[string]any{}).Optional(),
		field.String("error").Optional().Default(""),
		field.Time("started_at").Optional().Nillable(),
		field.Time("finished_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FlowJob) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("flow_jobs"),
	}
}

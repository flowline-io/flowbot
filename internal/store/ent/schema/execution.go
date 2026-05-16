package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Execution struct {
	ent.Schema
}

func (Execution) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("flow_id"),
		field.String("execution_id").NotEmpty(),
		field.String("trigger_type").NotEmpty(),
		field.String("trigger_id").Optional().Default(""),
		field.Int("state").Default(0),
		field.JSON("payload", map[string]any{}).Optional(),
		field.JSON("variables", map[string]any{}).Optional(),
		field.JSON("result", map[string]any{}).Optional(),
		field.String("error").Optional().Default(""),
		field.Time("started_at").Optional().Nillable(),
		field.Time("finished_at").Optional().Nillable(),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Execution) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("executions"),
	}
}

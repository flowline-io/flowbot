package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Job struct {
	ent.Schema
}

func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("workflow_id"),
		field.Int64("dag_id"),
		field.Int64("trigger_id"),
		field.Int32("script_version").Default(0),
		field.Int("state").Default(0),
		field.Time("started_at").Optional().Nillable(),
		field.Time("ended_at").Optional().Nillable(),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Job) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("steps", Step.Type),
	}
}

func (Job) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
		index.Fields("workflow_id"),
		index.Fields("state"),
	}
}

func (Job) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("jobs"),
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Review struct {
	ent.Schema
}

func (Review) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("objective_id").Optional(),
		field.Int32("type").Default(0),
		field.Int32("rating").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Review) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("objective", Objective.Type).Ref("reviews").Unique().Field("objective_id"),
		edge.To("evaluations", ReviewEvaluation.Type),
	}
}

func (Review) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (Review) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("reviews"),
	}
}

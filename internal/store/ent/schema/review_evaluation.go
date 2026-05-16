package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ReviewEvaluation struct {
	ent.Schema
}

func (ReviewEvaluation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("review_id").Optional(),
		field.String("question").NotEmpty(),
		field.String("reason").Default(""),
		field.String("solving").Default(""),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (ReviewEvaluation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("review", Review.Type).Ref("evaluations").Unique().Field("review_id"),
	}
}

func (ReviewEvaluation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
		index.Fields("review_id"),
	}
}

func (ReviewEvaluation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("review_evaluations"),
	}
}

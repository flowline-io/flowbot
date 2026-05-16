package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type KeyResult struct {
	ent.Schema
}

func (KeyResult) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("objective_id").Optional(),
		field.Int32("sequence").Default(0),
		field.String("title").NotEmpty(),
		field.String("memo").Default(""),
		field.Int32("initial_value").Default(0),
		field.Int32("target_value").Default(0),
		field.Int32("current_value").Default(0),
		field.String("value_mode").Default(""),
		field.String("tag").Default(""),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (KeyResult) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("objective", Objective.Type).Ref("key_results").Unique().Field("objective_id"),
		edge.To("key_result_values", KeyResultValue.Type),
		edge.To("todos", Todo.Type),
	}
}

func (KeyResult) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (KeyResult) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("key_results"),
	}
}

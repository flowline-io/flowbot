package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type KeyResultValue struct {
	ent.Schema
}

func (KeyResultValue) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("key_result_id").Optional(),
		field.Int32("value").Default(0),
		field.String("memo").Default(""),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (KeyResultValue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("key_result", KeyResult.Type).Ref("key_result_values").Unique().Field("key_result_id"),
	}
}

func (KeyResultValue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key_result_id"),
	}
}

func (KeyResultValue) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("key_result_values"),
	}
}

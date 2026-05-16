package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Todo struct {
	ent.Schema
}

func (Todo) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("key_result_id").Optional(),
		field.Int64("parent_id").Optional(),
		field.Int32("sequence").Default(0),
		field.String("content").NotEmpty(),
		field.String("category").Default(""),
		field.String("remark").Default(""),
		field.Int32("priority").Default(0),
		field.Int32("is_remind_at_time").Default(0),
		field.Int64("remind_at").Default(0),
		field.String("repeat_method").Default(""),
		field.String("repeat_rule").Default(""),
		field.Int64("repeat_end_at").Default(0),
		field.Int32("complete").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Todo) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("key_result", KeyResult.Type).Ref("todos").Unique().Field("key_result_id"),
		edge.To("sub_todos", Todo.Type).From("parent").Field("parent_id").Unique(),
	}
}

func (Todo) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("todos"),
	}
}

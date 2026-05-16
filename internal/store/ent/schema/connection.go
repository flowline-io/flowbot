package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Connection struct {
	ent.Schema
}

func (Connection) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),
		field.JSON("config", map[string]any{}),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Connection) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("connections"),
	}
}

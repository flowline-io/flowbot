package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Flow struct {
	ent.Schema
}

func (Flow) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("description").Optional().Default(""),
		field.Int("state").Default(0),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Flow) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("flows"),
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Bot struct {
	ent.Schema
}

func (Bot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty(),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Bot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("bots"),
	}
}

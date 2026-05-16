package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Platform struct {
	ent.Schema
}

func (Platform) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty(),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Platform) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("platforms"),
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Url struct {
	ent.Schema
}

func (Url) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty(),
		field.String("url").NotEmpty(),
		field.Int("state").Default(0),
		field.Int32("view_count").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Url) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("urls"),
	}
}

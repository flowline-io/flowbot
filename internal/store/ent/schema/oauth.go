package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OAuth struct {
	ent.Schema
}

func (OAuth) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),
		field.String("token").NotEmpty(),
		field.JSON("extra", map[string]any{}),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (OAuth) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (OAuth) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("oauth"),
	}
}

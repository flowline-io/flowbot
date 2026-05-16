package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Instruct struct {
	ent.Schema
}

func (Instruct) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("no").NotEmpty(),
		field.String("uid").NotEmpty(),
		field.String("object").NotEmpty(),
		field.String("bot").NotEmpty(),
		field.String("flag").NotEmpty(),
		field.JSON("content", map[string]any{}),
		field.Int("priority").Default(2),
		field.Int("state").Default(0),
		field.Time("expire_at"),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Instruct) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid"),
		index.Fields("no"),
	}
}

func (Instruct) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("instruct"),
	}
}

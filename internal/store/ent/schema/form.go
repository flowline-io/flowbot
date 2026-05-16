package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Form struct {
	ent.Schema
}

func (Form) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("form_id").NotEmpty(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.JSON("schema", map[string]any{}),
		field.JSON("values", map[string]any{}).Optional(),
		field.JSON("extra", map[string]any{}).Optional(),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Form) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("form_id"),
		index.Fields("uid", "topic"),
	}
}

func (Form) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("form"),
	}
}

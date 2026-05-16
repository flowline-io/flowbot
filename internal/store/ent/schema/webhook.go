package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Webhook struct {
	ent.Schema
}

func (Webhook) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("flag").NotEmpty(),
		field.String("secret").NotEmpty(),
		field.Int32("trigger_count").Default(0),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Webhook) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("webhook"),
	}
}

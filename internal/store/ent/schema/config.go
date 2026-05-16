package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ConfigData struct {
	ent.Schema
}

func (ConfigData) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("key").NotEmpty(),
		field.JSON("value", map[string]any{}),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (ConfigData) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (ConfigData) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("configs"),
	}
}

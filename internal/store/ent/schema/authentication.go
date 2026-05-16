package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Authentication struct {
	ent.Schema
}

func (Authentication) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("connection_id").Optional().Nillable(),
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),
		field.JSON("credentials", map[string]any{}),
		field.Time("expires_at").Optional().Nillable(),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Authentication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("authentications"),
	}
}

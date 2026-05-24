package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
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
		field.String("refresh_token").Optional(),
		field.Time("expires_at").Optional(),
		field.String("token_type").Optional(),
		field.String("scope").Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OAuth) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
		index.Fields("type"),
	}
}

func (OAuth) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("oauth"),
	}
}

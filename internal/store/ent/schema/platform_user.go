package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlatformUser struct {
	ent.Schema
}

func (PlatformUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("platform_id"),
		field.Int64("user_id"),
		field.String("flag").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("email").NotEmpty(),
		field.String("avatar_url").NotEmpty(),
		field.Bool("is_bot").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PlatformUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform_id"),
		index.Fields("user_id"),
		index.Fields("flag"),
	}
}

func (PlatformUser) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("platform_users"),
	}
}

package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlatformChannelUser struct {
	ent.Schema
}

func (PlatformChannelUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("platform_id"),
		field.String("channel_flag").NotEmpty(),
		field.String("user_flag").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PlatformChannelUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform_id"),
		index.Fields("channel_flag"),
		index.Fields("user_flag"),
	}
}

func (PlatformChannelUser) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("platform_channel_users"),
	}
}

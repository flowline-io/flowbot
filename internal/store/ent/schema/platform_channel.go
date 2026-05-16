package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlatformChannel struct {
	ent.Schema
}

func (PlatformChannel) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("platform_id"),
		field.Int64("channel_id"),
		field.String("flag").NotEmpty().Default("0"),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (PlatformChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform_id"),
		index.Fields("channel_id"),
	}
}

func (PlatformChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("platform_channels"),
	}
}

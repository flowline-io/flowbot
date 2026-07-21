package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NotifyChannel struct {
	ent.Schema
}

func (NotifyChannel) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").Unique().NotEmpty(),
		field.String("protocol").NotEmpty(),
		field.String("uri").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Bool("is_default").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotifyChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("protocol"),
		index.Fields("enabled"),
		index.Fields("is_default"),
	}
}

func (NotifyChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notify_channels"),
	}
}

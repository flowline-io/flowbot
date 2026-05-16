package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlatformBot struct {
	ent.Schema
}

func (PlatformBot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("platform_id"),
		field.Int64("bot_id"),
		field.String("flag").NotEmpty().Default("0"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PlatformBot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform_id"),
		index.Fields("bot_id"),
	}
}

func (PlatformBot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("platform_bots"),
	}
}

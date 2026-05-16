package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Message struct {
	ent.Schema
}

func (Message) Mixin() []ent.Mixin {
	return []ent.Mixin{}
}

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("platform_id"),
		field.String("platform_msg_id").Default(""),
		field.String("topic").NotEmpty(),
		field.String("role").Default("user"),
		field.String("session").NotEmpty().Default(""),
		field.JSON("content", map[string]any{}).Optional(),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Optional().Nillable(),
	}
}

func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("topic"),
		index.Fields("platform_id"),
	}
}

func (Message) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("messages"),
	}
}

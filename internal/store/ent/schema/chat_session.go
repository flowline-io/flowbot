package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatSession stores metadata for a direct-message chat assistant session.
type ChatSession struct {
	ent.Schema
}

func (ChatSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("uid").NotEmpty(),
		field.String("leaf_id").Default(""),
		field.Int("state").Default(0),
		field.String("mode").Default(string(ChatSessionModeNormal)),
		field.String("model").Default(""),
		field.String("thinking_level").Default(""),
		field.String("title").Default(""),
		field.String("preview").Default(""),
		field.Bool("pinned").Default(false),
		field.Bool("archived").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ChatSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid"),
		index.Fields("uid", "archived"),
		index.Fields("uid", "pinned"),
	}
}

func (ChatSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("chat_sessions"),
	}
}

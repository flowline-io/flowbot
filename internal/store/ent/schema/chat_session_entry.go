package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatSessionEntry stores one append-only node in a chat agent session tree.
type ChatSessionEntry struct {
	ent.Schema
}

func (ChatSessionEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("session_id").NotEmpty(),
		field.String("parent_id").Default(""),
		field.String("entry_type").NotEmpty(),
		field.JSON("payload", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (ChatSessionEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "created_at"),
	}
}

func (ChatSessionEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("chat_session_entries"),
	}
}

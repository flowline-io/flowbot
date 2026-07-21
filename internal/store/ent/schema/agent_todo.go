package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentTodo stores one checklist item for a chat agent session.
type AgentTodo struct {
	ent.Schema
}

func (AgentTodo) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("session_id").NotEmpty(),
		field.String("item_id").NotEmpty(),
		field.Text("content").NotEmpty(),
		field.String("status").NotEmpty(),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AgentTodo) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("session_id", "item_id").Unique(),
	}
}

func (AgentTodo) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_todos"),
	}
}

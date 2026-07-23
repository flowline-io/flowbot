package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentMemoryFact stores one keyed memory fact for a memory scope.
type AgentMemoryFact struct {
	ent.Schema
}

// Fields of the AgentMemoryFact.
func (AgentMemoryFact) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("scope").NotEmpty(),
		field.String("key").NotEmpty(),
		field.Text("value").NotEmpty(),
		field.Bool("pinned").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the AgentMemoryFact.
func (AgentMemoryFact) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope", "key").Unique(),
		index.Fields("scope", "pinned", "updated_at"),
	}
}

// Annotations of the AgentMemoryFact.
func (AgentMemoryFact) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_memory_facts"),
	}
}

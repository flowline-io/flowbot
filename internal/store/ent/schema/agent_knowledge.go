package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentKnowledge stores a markdown knowledge-base document for chat agents.
type AgentKnowledge struct {
	ent.Schema
}

// Fields of the AgentKnowledge.
func (AgentKnowledge) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("path").NotEmpty().Unique(),
		field.String("title").NotEmpty(),
		field.JSON("tags", []string{}).Default([]string{}),
		field.Text("summary").Default(""),
		field.Text("content").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the AgentKnowledge.
func (AgentKnowledge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("updated_at"),
		index.Fields("title"),
	}
}

// Annotations of the AgentKnowledge.
func (AgentKnowledge) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_knowledge"),
	}
}

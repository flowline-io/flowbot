package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSubagent stores a chat assistant subagent definition delegated through the task tool.
type AgentSubagent struct {
	ent.Schema
}

// Fields defines the subagent columns mirroring the agent skill schema with delegation-specific attributes.
func (AgentSubagent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
		field.Text("system_prompt").NotEmpty(),
		field.JSON("tools", []string{}).Optional(),
		field.String("model").Default(""),
		field.String("source").Default("global"),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes enforces a unique subagent name and an enabled lookup index.
func (AgentSubagent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
		index.Fields("enabled"),
	}
}

// Annotations pins the database table name for the subagent definitions.
func (AgentSubagent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_subagents"),
	}
}

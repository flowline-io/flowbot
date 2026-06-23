package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSkillFile stores an auxiliary file belonging to an agent skill directory.
type AgentSkillFile struct {
	ent.Schema
}

// Fields defines the agent skill file columns.
func (AgentSkillFile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("skill_flag").NotEmpty(),
		field.String("path").NotEmpty(),
		field.Text("content").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes enforces unique paths per skill and lookup by skill flag.
func (AgentSkillFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("skill_flag", "path").Unique(),
		index.Fields("skill_flag"),
	}
}

// Annotations pins the database table name for agent skill files.
func (AgentSkillFile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_skill_files"),
	}
}

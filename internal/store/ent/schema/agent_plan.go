package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentPlan stores a plan document produced during chat agent plan mode.
type AgentPlan struct {
	ent.Schema
}

func (AgentPlan) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("session_id").NotEmpty(),
		field.String("title").NotEmpty(),
		field.Text("content").NotEmpty(),
		field.String("source_entry_id").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AgentPlan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
	}
}

func (AgentPlan) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_plans"),
	}
}

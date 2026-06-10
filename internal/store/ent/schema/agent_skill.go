package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSkill stores an agent skill definition loaded into chat assistant prompts.
type AgentSkill struct {
	ent.Schema
}

func (AgentSkill) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
		field.Text("content").NotEmpty(),
		field.String("base_dir").Default(""),
		field.String("source").Default("global"),
		field.Bool("enabled").Default(true),
		field.Bool("disable_model_invocation").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AgentSkill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
		index.Fields("enabled"),
	}
}

func (AgentSkill) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_skills"),
	}
}

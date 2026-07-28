package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeActionSpec stores action-specific planning fields for a plan node.
type LifeActionSpec struct {
	ent.Schema
}

// Fields of the LifeActionSpec.
func (LifeActionSpec) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("plan_node_id").Unique(),
		field.String("task_type").Default("todo"),
		field.String("tracking_mode").Default("completion"),
		field.Bool("is_repeatable").Default(false),
		field.String("repeat_trigger").Default("none"),
		field.String("suggested_cadence").Default(""),
		field.Bool("is_identity_building").Default(false),
		field.Text("reason").Default(""),
		field.Bool("needs_user_confirmation").Default(false),
		field.String("difficulty").Default("C"),
		field.Int("base_exp_reward").Default(25),
		field.Int("base_gold_reward").Default(8),
		field.Time("confirmed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeActionSpec.
func (LifeActionSpec) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_type"),
		index.Fields("needs_user_confirmation"),
	}
}

// Annotations of the LifeActionSpec.
func (LifeActionSpec) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_action_specs"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeActionOccurrence is one executable todo or recurring instance.
type LifeActionOccurrence struct {
	ent.Schema
}

// Fields of the LifeActionOccurrence.
func (LifeActionOccurrence) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("plan_node_id"),
		field.String("kind").Default("one_time"),
		field.String("state").Default("pending"),
		field.Time("due_at"),
		field.Time("completed_at").Optional().Nillable(),
		field.String("cadence_snapshot").Default(""),
		field.Int64("source_occurrence_id").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeActionOccurrence.
func (LifeActionOccurrence) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "state"),
		index.Fields("plan_node_id"),
		index.Fields("plan_node_id", "due_at"),
	}
}

// Annotations of the LifeActionOccurrence.
func (LifeActionOccurrence) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_action_occurrences"),
	}
}

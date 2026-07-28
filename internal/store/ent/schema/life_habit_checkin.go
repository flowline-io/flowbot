package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeHabitCheckin is one daily habit check-in.
type LifeHabitCheckin struct {
	ent.Schema
}

// Fields of the LifeHabitCheckin.
func (LifeHabitCheckin) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("plan_node_id"),
		field.Time("checkin_date"),
		field.String("status").Default("done"),
		field.Text("note").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeHabitCheckin.
func (LifeHabitCheckin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("plan_node_id"),
		index.Fields("plan_node_id", "checkin_date").Unique(),
	}
}

// Annotations of the LifeHabitCheckin.
func (LifeHabitCheckin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_habit_checkins"),
	}
}

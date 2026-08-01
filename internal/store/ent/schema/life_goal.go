package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeGoal is a PARA-style long-term goal.
type LifeGoal struct {
	ent.Schema
}

// Fields of the LifeGoal.
func (LifeGoal) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.String("title").NotEmpty(),
		field.String("category").Default("Project"),
		field.String("status").Default("Active"),
		field.Int64("area_id").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeGoal.
func (LifeGoal) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "status"),
		index.Fields("area_id"),
	}
}

// Annotations of the LifeGoal.
func (LifeGoal) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_goals"),
	}
}

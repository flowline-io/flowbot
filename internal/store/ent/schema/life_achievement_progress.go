package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeAchievementProgress stores per-profile counts keyed by achievement condition.
type LifeAchievementProgress struct {
	ent.Schema
}

// Fields of the LifeAchievementProgress.
func (LifeAchievementProgress) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.String("condition_key").NotEmpty(),
		field.Int("current_count").Default(0),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeAchievementProgress.
func (LifeAchievementProgress) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "condition_key").Unique(),
	}
}

// Annotations of the LifeAchievementProgress.
func (LifeAchievementProgress) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_achievement_progress"),
	}
}

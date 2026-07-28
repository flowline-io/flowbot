package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeAchievementUnlock records that a profile unlocked a catalog achievement.
type LifeAchievementUnlock struct {
	ent.Schema
}

// Fields of the LifeAchievementUnlock.
func (LifeAchievementUnlock) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.String("achievement_flag").NotEmpty(),
		field.Time("unlocked_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeAchievementUnlock.
func (LifeAchievementUnlock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "achievement_flag").Unique(),
	}
}

// Annotations of the LifeAchievementUnlock.
func (LifeAchievementUnlock) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_achievement_unlocks"),
	}
}

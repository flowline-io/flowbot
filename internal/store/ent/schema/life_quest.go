package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeQuest is a one-time, daily, or boss quest.
type LifeQuest struct {
	ent.Schema
}

// Fields of the LifeQuest.
func (LifeQuest) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("goal_id").Optional().Nillable(),
		field.Int64("skill_id"),
		field.String("title").NotEmpty(),
		field.Text("prompt").Default(""),
		field.String("type").Default("One-Time"),
		field.String("ai_evaluated_difficulty").Default("E"),
		field.Int("ai_evaluated_fear").Default(1),
		field.Int("base_exp_reward").Default(10),
		field.Int("base_gold_reward").Default(5),
		field.String("drop_tier").Default("Common"),
		field.String("status").Default("Pending"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("completed_at").Optional().Nillable(),
	}
}

// Indexes of the LifeQuest.
func (LifeQuest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "status"),
		index.Fields("skill_id"),
	}
}

// Annotations of the LifeQuest.
func (LifeQuest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_quests"),
	}
}

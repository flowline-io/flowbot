package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeAIContext stores AI dungeon-master context for a profile.
type LifeAIContext struct {
	ent.Schema
}

// Fields of the LifeAIContext.
func (LifeAIContext) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("life_profile_id").Unique(),
		field.Float("historical_completion_rate").Default(0),
		field.JSON("recent_mood_and_burnout", map[string]any{}).Optional(),
		field.String("ai_dm_personality").Default("gentle guide"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeAIContext.
func (LifeAIContext) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id").Unique(),
	}
}

// Annotations of the LifeAIContext.
func (LifeAIContext) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_ai_contexts"),
	}
}

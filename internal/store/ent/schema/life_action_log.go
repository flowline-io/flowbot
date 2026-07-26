package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeActionLog records quest completion and loot rolls.
type LifeActionLog struct {
	ent.Schema
}

// Fields of the LifeActionLog.
func (LifeActionLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("quest_id"),
		field.Int("gained_exp").Default(0),
		field.Int("gained_gold").Default(0),
		field.Int64("dropped_inventory_id").Optional().Nillable(),
		field.Float("dice_roll_result").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeActionLog.
func (LifeActionLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("quest_id"),
	}
}

// Annotations of the LifeActionLog.
func (LifeActionLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_action_logs"),
	}
}

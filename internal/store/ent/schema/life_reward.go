package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeReward is a player-defined real-life reward redeemable with gold.
type LifeReward struct {
	ent.Schema
}

// Fields of the LifeReward.
func (LifeReward) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.String("name").NotEmpty(),
		field.String("notes").Default(""),
		field.Int("price"),
		field.Int("cooldown_hours").Default(0),
		field.Bool("active").Default(true),
		field.Time("last_redeemed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeReward.
func (LifeReward) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "active"),
	}
}

// Annotations of the LifeReward.
func (LifeReward) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_rewards"),
	}
}

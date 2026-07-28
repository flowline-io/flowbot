package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeRewardRedemption is an audit row for one honor-system reward redeem.
type LifeRewardRedemption struct {
	ent.Schema
}

// Fields of the LifeRewardRedemption.
func (LifeRewardRedemption) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("life_reward_id"),
		field.String("reward_name").NotEmpty(),
		field.Int("price_paid"),
		field.Time("redeemed_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeRewardRedemption.
func (LifeRewardRedemption) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "redeemed_at"),
		index.Fields("life_reward_id"),
	}
}

// Annotations of the LifeRewardRedemption.
func (LifeRewardRedemption) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_reward_redemptions"),
	}
}

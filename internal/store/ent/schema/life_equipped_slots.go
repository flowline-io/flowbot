package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeEquippedSlots is the wear-state truth for one profile.
type LifeEquippedSlots struct {
	ent.Schema
}

// Fields of the LifeEquippedSlots.
func (LifeEquippedSlots) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("life_profile_id").Unique(),
		field.Int64("head_slot").Optional().Nillable(),
		field.Int64("weapon_slot").Optional().Nillable(),
		field.Int64("armor_slot").Optional().Nillable(),
		field.Int64("shoes_slot").Optional().Nillable(),
		field.Int64("accessory_slot").Optional().Nillable(),
		field.Int64("artifact_slot").Optional().Nillable(),
		field.Time("tarnished_until").Optional().Nillable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeEquippedSlots.
func (LifeEquippedSlots) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id").Unique(),
	}
}

// Annotations of the LifeEquippedSlots.
func (LifeEquippedSlots) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_equipped_slots"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeInventory is a profile-owned equipment instance.
type LifeInventory struct {
	ent.Schema
}

// Fields of the LifeInventory.
func (LifeInventory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("equipment_id"),
		field.Int64("source_quest_id").Optional().Nillable(),
		field.String("instance_name").Default(""),
		field.Text("instance_lore").Default(""),
		field.JSON("instance_buffs", map[string]any{}).Optional(),
		field.String("lore_status").Default("none"),
		field.Time("tarnished_until").Optional().Nillable(),
		field.Time("acquired_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeInventory.
func (LifeInventory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("equipment_id"),
		index.Fields("lore_status"),
	}
}

// Annotations of the LifeInventory.
func (LifeInventory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_inventories"),
	}
}

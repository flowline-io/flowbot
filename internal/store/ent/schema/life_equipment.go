package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeEquipment is a catalog equipment template.
type LifeEquipment struct {
	ent.Schema
}

// Fields of the LifeEquipment.
func (LifeEquipment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("name").NotEmpty(),
		field.String("rarity").Default("Common"),
		field.String("slot_type").NotEmpty(),
		field.JSON("stat_buffs", map[string]any{}).Optional(),
		field.JSON("ai_unlocked_privilege", map[string]any{}).Optional(),
		field.Text("ai_lore_text").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeEquipment.
func (LifeEquipment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("rarity"),
		index.Fields("slot_type"),
	}
}

// Annotations of the LifeEquipment.
func (LifeEquipment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_equipments"),
	}
}

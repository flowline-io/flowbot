package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeLootTable configures drop chance and equipment pool per tier.
type LifeLootTable struct {
	ent.Schema
}

// Fields of the LifeLootTable.
func (LifeLootTable) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("drop_tier").NotEmpty().Unique(),
		field.Float("base_drop_chance").Default(0.2),
		field.JSON("item_pool_flags", []string{}).Optional(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeLootTable.
func (LifeLootTable) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("drop_tier").Unique(),
	}
}

// Annotations of the LifeLootTable.
func (LifeLootTable) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_loot_tables"),
	}
}

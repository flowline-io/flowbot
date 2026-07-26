package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeSkill is a secondary skill under a characteristic.
type LifeSkill struct {
	ent.Schema
}

// Fields of the LifeSkill.
func (LifeSkill) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("characteristic_id"),
		field.String("name").NotEmpty(),
		field.Int("level").Default(1),
		field.Int64("current_exp").Default(0),
		field.Float("exp_to_characteristic_ratio").Default(0.5),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeSkill.
func (LifeSkill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("characteristic_id"),
		index.Fields("life_profile_id", "name").Unique(),
	}
}

// Annotations of the LifeSkill.
func (LifeSkill) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_skills"),
	}
}

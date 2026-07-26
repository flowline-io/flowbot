package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeProfile is the solo RPG profile for one Flowbot operator.
type LifeProfile struct {
	ent.Schema
}

// Fields of the LifeProfile.
func (LifeProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("user_id").NotEmpty().Unique(),
		field.String("nickname").Default(""),
		field.Int("level").Default(1),
		field.Int64("exp").Default(0),
		field.Int("gold").Default(0),
		field.String("class_type").Default("Architect"),
		field.Float("base_drop_rate_bonus").Default(0),
		field.JSON("pity_by_tier", map[string]int{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeProfile.
func (LifeProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").Unique(),
	}
}

// Annotations of the LifeProfile.
func (LifeProfile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_profiles"),
	}
}

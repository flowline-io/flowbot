package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeCharacteristic is a primary stat (INT/PHY/WIL/CHA) for a life profile.
type LifeCharacteristic struct {
	ent.Schema
}

// Fields of the LifeCharacteristic.
func (LifeCharacteristic) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.String("name").NotEmpty(),
		field.String("code").NotEmpty(),
		field.Int("level").Default(1),
		field.Int64("current_exp").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeCharacteristic.
func (LifeCharacteristic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "code").Unique(),
	}
}

// Annotations of the LifeCharacteristic.
func (LifeCharacteristic) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_characteristics"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeAchievement is a seeded memorial achievement catalog entry.
type LifeAchievement struct {
	ent.Schema
}

// Fields of the LifeAchievement.
func (LifeAchievement) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("name").NotEmpty(),
		field.Text("description").Default(""),
		field.Bool("active").Default(true),
		field.String("kind").Default("first"),
		field.String("quest_type").Default(""),
		field.String("difficulty").Default(""),
		field.Int("threshold").Default(1),
		field.Int("sort_order").Default(0),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifeAchievement.
func (LifeAchievement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("active"),
		index.Fields("sort_order"),
	}
}

// Annotations of the LifeAchievement.
func (LifeAchievement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_achievements"),
	}
}

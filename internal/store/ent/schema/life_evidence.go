package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeEvidence stores user-submitted proof for a quest.
type LifeEvidence struct {
	ent.Schema
}

// Fields of the LifeEvidence.
func (LifeEvidence) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("quest_id").Optional().Nillable(),
		field.String("source_type").Default("note"),
		field.Text("content").Default(""),
		field.String("source_url").Default(""),
		field.String("summary").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeEvidence.
func (LifeEvidence) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("quest_id"),
		index.Fields("life_profile_id", "quest_id"),
	}
}

// Annotations of the LifeEvidence.
func (LifeEvidence) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_evidence"),
	}
}

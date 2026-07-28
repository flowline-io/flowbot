package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeAdjudication stores a suggested AI ruling for a quest.
type LifeAdjudication struct {
	ent.Schema
}

// Fields of the LifeAdjudication.
func (LifeAdjudication) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("quest_id"),
		field.String("status").Default("suggested"),
		field.String("verdict").Default("needs_more_evidence"),
		field.Text("reason").Default(""),
		field.Int("suggested_exp").Default(0),
		field.Int("suggested_gold").Default(0),
		field.JSON("suggested_next_steps", []string{}).Default([]string{}),
		field.JSON("evidence_snapshot", []map[string]any{}).Default([]map[string]any{}),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("applied_at").Optional().Nillable(),
	}
}

// Indexes of the LifeAdjudication.
func (LifeAdjudication) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("quest_id"),
		index.Fields("quest_id", "status"),
		index.Fields("life_profile_id", "quest_id"),
	}
}

// Annotations of the LifeAdjudication.
func (LifeAdjudication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_adjudications"),
	}
}

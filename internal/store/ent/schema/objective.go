package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Objective struct {
	ent.Schema
}

func (Objective) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int32("sequence").Default(0),
		field.Int32("progress").Default(0),
		field.String("title").NotEmpty(),
		field.String("memo").Default(""),
		field.String("motive").Default(""),
		field.String("feasibility").Default(""),
		field.Int32("is_plan").Default(0),
		field.Time("plan_start"),
		field.Time("plan_end"),
		field.Int32("total_value").Default(0),
		field.Int32("current_value").Default(0),
		field.String("tag").Default(""),
		field.Time("created_data").Immutable().Default(time.Now),
		field.Time("updated_date").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Objective) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("key_results", KeyResult.Type),
		edge.To("reviews", Review.Type),
	}
}

func (Objective) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (Objective) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("objectives"),
	}
}

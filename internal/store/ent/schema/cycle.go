package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Cycle struct {
	ent.Schema
}

func (Cycle) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("objectives").NotEmpty(),
		field.Time("start_date"),
		field.Time("end_date"),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Cycle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (Cycle) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("cycles"),
	}
}

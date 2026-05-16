package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type CounterRecord struct {
	ent.Schema
}

func (CounterRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("counter_id"),
		field.Int32("digit").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (CounterRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("counter_records"),
	}
}

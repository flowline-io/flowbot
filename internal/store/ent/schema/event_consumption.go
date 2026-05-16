package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EventConsumption struct {
	ent.Schema
}

func (EventConsumption) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("consumer_name").NotEmpty(),
		field.String("event_id").NotEmpty(),
		field.Time("created_at").Immutable(),
	}
}

func (EventConsumption) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("consumer_name", "event_id").Unique(),
	}
}

func (EventConsumption) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("event_consumptions"),
	}
}

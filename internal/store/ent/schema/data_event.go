package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type DataEvent struct {
	ent.Schema
}

func (DataEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("event_id").NotEmpty().Unique(),
		field.String("event_type").NotEmpty(),
		field.String("source").Default(""),
		field.String("capability").Default(""),
		field.String("operation").Default(""),
		field.String("backend").Default(""),
		field.String("app").Default(""),
		field.String("entity_id").Default(""),
		field.String("idempotency_key").Default(""),
		field.String("uid").Default(""),
		field.String("topic").Default(""),
		field.JSON("data", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (DataEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
	}
}

func (DataEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("data_events"),
	}
}

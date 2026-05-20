package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type EventOutbox struct {
	ent.Schema
}

func (EventOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("event_id").NotEmpty(),
		field.JSON("payload", map[string]any{}),
		field.Bool("published").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (EventOutbox) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("event_outbox"),
	}
}

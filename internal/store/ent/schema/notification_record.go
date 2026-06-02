package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NotificationRecord struct {
	ent.Schema
}

func (NotificationRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("channel").NotEmpty(),
		field.String("template_id").NotEmpty(),
		field.String("summary").Default(""),
		field.Enum("status").Values("success", "failed", "dropped", "throttled", "aggregated", "muted").Default("success"),
		field.String("error_msg").Default(""),
		field.JSON("payload_snapshot", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (NotificationRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "id"),
		index.Fields("uid", "created_at"),
	}
}

func (NotificationRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notification_records"),
	}
}

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
		field.String("rule_id").Default(""),
		field.String("summary").Default(""),
		field.Enum("status").Values(
			"success", "failed", "dropped", "throttled", "aggregated", "muted",
			"deferred", "cancelled",
		).Default("success"),
		field.String("error_msg").Default(""),
		field.JSON("payload_snapshot", map[string]any{}).Optional(),
		field.String("correlation_id").Default(""),
		field.Time("escalate_at").Optional().Nillable(),
		field.Time("read_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (NotificationRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "id"),
		index.Fields("uid", "created_at"),
		index.Fields("uid", "channel", "id"),
		index.Fields("uid", "rule_id", "id"),
		index.Fields("uid", "read_at"),
		index.Fields("uid", "correlation_id"),
		index.Fields("uid", "channel", "read_at"),
		index.Fields("status", "escalate_at"),
	}
}

func (NotificationRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notification_records"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NotifyRule struct {
	ent.Schema
}

func (NotifyRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("rule_id").Unique().NotEmpty(),
		field.String("name").NotEmpty(),
		field.Enum("action").Values("throttle", "aggregate", "mute", "drop"),
		field.String("event_pattern").Default("*").NotEmpty(),
		field.String("channel_pattern").Default("*").NotEmpty(),
		field.String("condition").Optional(),
		field.Int("priority").Default(0),
		field.JSON("params", map[string]any{}),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotifyRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("priority"),
		index.Fields("enabled"),
	}
}

func (NotifyRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notify_rules"),
	}
}

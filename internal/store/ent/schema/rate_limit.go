package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type RateLimit struct {
	ent.Schema
}

func (RateLimit) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("flow_id").Optional().Nillable(),
		field.String("node_id").Optional().Default(""),
		field.String("limit_type").NotEmpty(),
		field.Int("limit_value").Default(0),
		field.Int("window_size").Default(60),
		field.String("window_unit").Default("second"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RateLimit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("rate_limits"),
	}
}

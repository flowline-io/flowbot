package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// GatewayWorker tracks last-seen heartbeats from cmd/gateway workers.
type GatewayWorker struct {
	ent.Schema
}

// Fields of the GatewayWorker.
func (GatewayWorker) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("worker_id").NotEmpty().Unique().Immutable(),
		field.Time("last_seen_at").Default(time.Now),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Annotations pins the database table name.
func (GatewayWorker) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("gateway_workers"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/flowline-io/flowbot/pkg/types"
)

// IdempotencyRecord is one write claim for an access token, operation, and client key.
// Rows are not expired. in_progress left by a stopped process is moved to unknown at startup.
type IdempotencyRecord struct {
	ent.Schema
}

// Fields of the IdempotencyRecord.
func (IdempotencyRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("token_id"),
		field.String("operation").NotEmpty().MaxLen(512),
		field.String("idempotency_key").NotEmpty().MaxLen(255),
		field.String("param_hash").NotEmpty().MaxLen(64),
		field.Enum("status").
			Values(string(types.IdempotencyInProgress), string(types.IdempotencySucceeded), string(types.IdempotencyUnknown)).
			Default(string(types.IdempotencyInProgress)),
		field.Text("result_json").Optional().Nillable(),
		field.String("correlation_id").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes enforces one claim per token, operation, and key.
func (IdempotencyRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_id", "operation", "idempotency_key").Unique(),
	}
}

// Annotations pins the database table name.
func (IdempotencyRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("idempotency_records"),
	}
}

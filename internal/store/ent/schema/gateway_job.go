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

// GatewayJob records one local-CLI gateway job claimed by cmd/gateway workers.
type GatewayJob struct {
	ent.Schema
}

// Fields of the GatewayJob.
func (GatewayJob) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("job_id").NotEmpty().Unique().Immutable(),
		field.String("uid").Default(""),
		field.String("cli").NotEmpty(),
		field.Text("prompt").NotEmpty(),
		field.String("cwd").Default(""),
		field.String("status").Default(string(types.GatewayJobPending)),
		field.Text("output").Default(""),
		field.Int("exit_code").Optional().Nillable(),
		field.Text("error_text").Default(""),
		field.Bool("truncated").Default(false),
		field.Int64("duration_ms").Default(0),
		field.String("worker_id").Default(""),
		field.Time("lease_until").Optional().Nillable(),
		field.Time("claimed_at").Optional().Nillable(),
		field.Time("finished_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes supports claim and lease reclaim lookups.
func (GatewayJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("status", "lease_until"),
		index.Fields("uid"),
	}
}

// Annotations pins the database table name.
func (GatewayJob) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("gateway_jobs"),
	}
}

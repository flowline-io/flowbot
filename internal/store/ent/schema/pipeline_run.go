package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PipelineRun struct {
	ent.Schema
}

func (PipelineRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("pipeline_name").NotEmpty(),
		field.String("event_id").NotEmpty().Unique(),
		field.String("event_type").Default(""),
		field.Enum("trigger_source").
			Values("event", "webhook", "cron", "manual").
			Default("event"),
		field.Int("status").Default(0),
		field.String("error").Optional().Default(""),
		field.JSON("checkpoint_data", map[string]any{}).Optional(),
		field.Time("last_heartbeat").Optional().Nillable(),
		field.Time("started_at"),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (PipelineRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("pipeline_name"),
	}
}

func (PipelineRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_runs"),
	}
}

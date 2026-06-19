package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatScheduledTask stores a user-owned cron or one-shot chat agent job.
type ChatScheduledTask struct {
	ent.Schema
}

func (ChatScheduledTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("uid").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("schedule_kind").NotEmpty(),
		field.String("cron").Default(""),
		field.Time("run_at").Optional().Nillable(),
		field.Text("prompt").NotEmpty(),
		field.JSON("delivery", map[string]any{}).Optional(),
		field.String("source_session_id").Default(""),
		field.String("state").Default(string(ChatScheduledTaskStateActive)),
		field.Time("last_run_at").Optional().Nillable(),
		field.Time("next_run_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ChatScheduledTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid"),
		index.Fields("state"),
		index.Fields("schedule_kind", "state"),
	}
}

func (ChatScheduledTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("chat_scheduled_tasks"),
	}
}

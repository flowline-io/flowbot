package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatScheduledTaskRun records one execution of a chat scheduled task.
type ChatScheduledTaskRun struct {
	ent.Schema
}

func (ChatScheduledTaskRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.String("task_id").NotEmpty(),
		field.String("run_session_id").NotEmpty(),
		field.String("state").Default(string(ChatScheduledTaskRunStateRunning)),
		field.Text("reply").Default(""),
		field.Text("error").Default(""),
		field.Time("started_at").Immutable().Default(time.Now),
		field.Time("finished_at").Optional().Nillable(),
	}
}

func (ChatScheduledTaskRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("task_id", "started_at"),
	}
}

func (ChatScheduledTaskRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("chat_scheduled_task_runs"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSubagentTaskStatus is the lifecycle state of a delegated subagent task.
type AgentSubagentTaskStatus string

const (
	// AgentSubagentTaskStatusRunning indicates the subagent is still executing.
	AgentSubagentTaskStatusRunning AgentSubagentTaskStatus = "running"
	// AgentSubagentTaskStatusCompleted indicates the subagent finished successfully.
	AgentSubagentTaskStatusCompleted AgentSubagentTaskStatus = "completed"
	// AgentSubagentTaskStatusFailed indicates the subagent run failed.
	AgentSubagentTaskStatusFailed AgentSubagentTaskStatus = "failed"
)

// AgentSubagentTask records one task delegation through the delegate_subagent tool.
type AgentSubagentTask struct {
	ent.Schema
}

// Fields defines the subagent task columns.
func (AgentSubagentTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("session_id").NotEmpty(),
		field.String("subagent_name").NotEmpty(),
		field.String("description").Default(""),
		field.Text("prompt").NotEmpty(),
		field.String("status").Default(string(AgentSubagentTaskStatusRunning)),
		field.Text("result").Default(""),
		field.Text("error_text").Default(""),
		field.Int("depth").Default(0),
		field.Time("started_at").Immutable().Default(time.Now),
		field.Time("finished_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes supports session and status lookups for task history.
func (AgentSubagentTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}

// Annotations pins the database table name for subagent task records.
func (AgentSubagentTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_subagent_tasks"),
	}
}

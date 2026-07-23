package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSessionSummaryStatus values for session summary rows.
const (
	AgentSessionSummaryPending = "pending"
	AgentSessionSummaryReady   = "ready"
	AgentSessionSummaryFailed  = "failed"
)

// AgentSessionSummary stores a cross-session searchable summary for one chat session.
type AgentSessionSummary struct {
	ent.Schema
}

// Fields of the AgentSessionSummary.
func (AgentSessionSummary) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("session_flag").NotEmpty().Unique(),
		field.String("scope").NotEmpty(),
		field.String("title").Default(""),
		field.Text("summary").Default(""),
		field.String("status").NotEmpty().Default(AgentSessionSummaryPending),
		field.Text("error").Default(""),
		field.String("claim_token").Default(""),
		field.Time("claimed_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the AgentSessionSummary.
func (AgentSessionSummary) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope", "status", "updated_at"),
		index.Fields("status", "claimed_at"),
	}
}

// Annotations of the AgentSessionSummary.
func (AgentSessionSummary) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agent_session_summaries"),
	}
}

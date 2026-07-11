package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LLMUsageRecord stores token consumption for one LLM call.
type LLMUsageRecord struct {
	ent.Schema
}

func (LLMUsageRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("session_id").Default(""),
		field.String("model").Default(""),
		field.Int("prompt_tokens").Default(0),
		field.Int("completion_tokens").Default(0),
		field.Int("total_tokens").Default(0),
		field.Int("cache_read").Default(0),
		field.Int("cache_write").Default(0),
		field.String("source").Default("agent"),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (LLMUsageRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "created_at"),
		index.Fields("uid", "model", "created_at"),
	}
}

func (LLMUsageRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("llm_usage_records"),
	}
}

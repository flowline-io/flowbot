package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FunctionRun records one invocation of a named function.
type FunctionRun struct {
	ent.Schema
}

// Fields of the FunctionRun.
func (FunctionRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("function_name").NotEmpty(),
		field.Int("version"),
		field.Enum("status").Values("running", "succeeded", "failed").Default("running"),
		field.Int64("duration_ms").Default(0),
		field.Int("exit_code").Optional().Nillable(),
		field.Text("error").Default(""),
		field.Text("result_json").Optional().Nillable(),
		// Nillable so NULL keys do not collide under unique (function_name, idempotency_key)
		// on Postgres and SQLite.
		field.String("idempotency_key").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes supports list-by-name and idempotent replay lookups.
func (FunctionRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("function_name"),
		index.Fields("function_name", "idempotency_key").Unique(),
	}
}

// Annotations pins the database table name.
func (FunctionRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("function_runs"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FunctionDefinitionVersion stores immutable published snapshots of a named function.
type FunctionDefinitionVersion struct {
	ent.Schema
}

// Fields of the FunctionDefinitionVersion.
func (FunctionDefinitionVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("function_name").NotEmpty(),
		field.Int("version"),
		field.Text("metadata").NotEmpty(),
		field.String("entrypoint").NotEmpty(),
		field.Text("source").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes enforces one snapshot per (function_name, version).
func (FunctionDefinitionVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("function_name", "version").Unique(),
	}
}

// Annotations pins the database table name.
func (FunctionDefinitionVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("function_definition_versions"),
	}
}

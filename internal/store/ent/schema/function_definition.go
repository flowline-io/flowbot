package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// FunctionDefinition is a named function with draft and published snapshots.
type FunctionDefinition struct {
	ent.Schema
}

// Fields of the FunctionDefinition.
func (FunctionDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique().
			Comment("function name, must match ^[\\p{L}\\p{N}][\\p{L}\\p{N}_-]*$").
			Match(PipelineNamePattern),
		field.Text("metadata_draft").Default(""),
		field.String("entrypoint_draft").Default(""),
		field.Text("source_draft").Default(""),
		field.Text("metadata_published").Optional().Nillable(),
		field.String("entrypoint_published").Optional().Nillable(),
		field.Text("source_published").Optional().Nillable(),
		field.Int("version").Default(1),
		field.Enum("status").Values("draft", "published").Default("draft"),
		field.String("created_by").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Annotations pins the database table name.
func (FunctionDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("function_definitions"),
	}
}

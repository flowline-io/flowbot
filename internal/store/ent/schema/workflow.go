package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Workflow stores a normalized workflow definition (not a YAML blob).
type Workflow struct {
	ent.Schema
}

// Fields of the Workflow.
func (Workflow) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique().
			Comment("workflow name, must match ^[\\p{L}\\p{N}][\\p{L}\\p{N}_-]*$").
			Match(PipelineNamePattern),
		field.String("describe").Optional().Default(""),
		field.Bool("enabled").Default(true),
		field.Bool("resumable").Default(false),
		field.Int("max_concurrency").Default(1),
		field.JSON("inputs", []map[string]any{}).Optional(),
		field.JSON("pipeline", []string{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Annotations of the Workflow.
func (Workflow) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflows"),
	}
}

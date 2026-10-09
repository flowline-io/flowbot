package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PipelineDefinition struct {
	ent.Schema
}

// Fields of PipelineDefinition.
func (PipelineDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique().
			Comment("pipeline name, must match ^[\\p{L}\\p{N}][\\p{L}\\p{N}_-]*$").
			Match(PipelineNamePattern),
		field.String("description").Optional().Default(""),
		field.Text("yaml_draft").Default(""),
		field.Text("yaml_published").Optional().Nillable(),
		field.Int("version").Default(1),
		field.Enum("status").Values("draft", "published").Default("draft"),
		// CreatedBy is the Web UI user UID that created this pipeline (e.g. user-admin).
		field.String("created_by").Default(""),
		field.String("blueprint_source").Default(""),
		field.String("blueprint_id").Default(""),
		field.String("blueprint_hash").Default(""),
		field.Text("blueprint_yaml").Default(""),
		field.JSON("blueprint_inputs", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of PipelineDefinition.
func (PipelineDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("blueprint_source", "blueprint_id"),
	}
}

// Annotations of PipelineDefinition.
func (PipelineDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definitions"),
	}
}

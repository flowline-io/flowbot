package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PipelineDefinitionVersion stores YAML snapshots of pipeline definitions
// each time a pipeline is published. It enforces a composite unique constraint
// on (pipeline_name, version) so that each name+version pair is immutable.
type PipelineDefinitionVersion struct {
	ent.Schema
}

func (PipelineDefinitionVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("pipeline_name").NotEmpty(),
		field.Int("version"),
		field.Text("yaml").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (PipelineDefinitionVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("pipeline_name", "version").Unique(),
	}
}

func (PipelineDefinitionVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definition_versions"),
	}
}

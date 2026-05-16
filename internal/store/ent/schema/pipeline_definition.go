package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type PipelineDefinition struct {
	ent.Schema
}

func (PipelineDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique(),
		field.String("description").Optional().Default(""),
		field.Bool("enabled").Default(true),
		field.JSON("trigger", map[string]any{}).Optional(),
		field.JSON("steps", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PipelineDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definitions"),
	}
}

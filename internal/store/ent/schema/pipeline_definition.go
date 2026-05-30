package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type PipelineDefinition struct {
	ent.Schema
}

func (PipelineDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique().
			Comment("pipeline name, must match ^[a-zA-Z0-9][a-zA-Z0-9_-]*$").
			Match(regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)),
		field.String("description").Optional().Default(""),
		field.Text("yaml_draft").Default(""),
		field.Text("yaml_published").Optional().Nillable(),
		field.Int("version").Default(1),
		field.Enum("status").Values("draft", "published").Default("draft"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PipelineDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definitions"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// PipelineBlueprintTemplate is a user-imported pipeline blueprint in the template library.
type PipelineBlueprintTemplate struct {
	ent.Schema
}

// Fields of PipelineBlueprintTemplate.
func (PipelineBlueprintTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("blueprint_id").NotEmpty().Unique(),
		field.String("title").NotEmpty(),
		field.String("description").Optional().Default(""),
		field.Text("yaml").NotEmpty(),
		field.String("content_hash").NotEmpty(),
		field.String("created_by").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Annotations of PipelineBlueprintTemplate.
func (PipelineBlueprintTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_blueprint_templates"),
	}
}

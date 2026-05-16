package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type App struct {
	ent.Schema
}

func (App) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty(),
		field.String("path").NotEmpty(),
		field.String("container_id").Optional().Default(""),
		field.String("status").Default("unknown"),
		field.JSON("docker_info", map[string]any{}).Optional(),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (App) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("apps"),
	}
}

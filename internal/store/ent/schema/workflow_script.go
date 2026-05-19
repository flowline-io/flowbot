package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type WorkflowScript struct {
	ent.Schema
}

func (WorkflowScript) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id").Optional(),
		field.String("lang").NotEmpty(),
		field.String("code").NotEmpty(),
		field.Int32("version").Default(1),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WorkflowScript) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_script"),
	}
}

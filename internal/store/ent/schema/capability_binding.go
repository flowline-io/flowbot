package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type CapabilityBinding struct {
	ent.Schema
}

func (CapabilityBinding) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("capability").NotEmpty(),
		field.String("backend").NotEmpty(),
		field.String("app").NotEmpty(),
		field.Bool("healthy").Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CapabilityBinding) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("capability_bindings"),
	}
}

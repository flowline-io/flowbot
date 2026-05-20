// Package schema provides Ent ORM schema definitions.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Agent struct {
	ent.Schema
}

func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("hostid").NotEmpty(),
		field.String("hostname").NotEmpty(),
		field.Int32("online_duration").Default(0),
		field.Time("last_online_at"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Agent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agents"),
	}
}

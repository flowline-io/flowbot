package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
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
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Agent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("agents"),
	}
}

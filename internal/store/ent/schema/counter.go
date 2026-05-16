package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Counter struct {
	ent.Schema
}

func (Counter) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("flag").NotEmpty(),
		field.Int64("digit").Default(0),
		field.Int32("status").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Counter) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
	}
}

func (Counter) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("counters"),
	}
}

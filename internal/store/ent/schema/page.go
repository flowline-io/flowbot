package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Page struct {
	ent.Schema
}

func (Page) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("page_id").NotEmpty(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("type").NotEmpty(),
		field.JSON("schema", map[string]any{}),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Page) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("page_id"),
		index.Fields("uid", "topic"),
	}
}

func (Page) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pages"),
	}
}

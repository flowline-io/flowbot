package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type Topic struct {
	ent.Schema
}

func (Topic) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty(),
		field.String("platform").NotEmpty(),
		field.Int64("owner"),
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),
		field.String("tags").Optional().Default(""),
		field.Int("state").Default(0),
		field.Time("touched_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Topic) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("topics"),
	}
}

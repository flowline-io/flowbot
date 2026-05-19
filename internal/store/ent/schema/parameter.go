package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Parameter struct {
	ent.Schema
}

func (Parameter) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.JSON("params", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("expired_at"),
	}
}

func (Parameter) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("parameter"),
	}
}

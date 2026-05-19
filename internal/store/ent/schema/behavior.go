package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Behavior struct {
	ent.Schema
}

func (Behavior) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("flag").NotEmpty(),
		field.Int32("count").Default(0),
		field.JSON("extra", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Behavior) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid"),
		index.Fields("flag"),
	}
}

func (Behavior) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("behavior"),
	}
}

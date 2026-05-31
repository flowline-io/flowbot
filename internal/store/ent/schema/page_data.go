package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PageData represents a shared page with token-based access.
type PageData struct {
	ent.Schema
}

// Fields of the PageData.
func (PageData) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("token").NotEmpty().Unique(),
		field.String("type").NotEmpty(),
		field.String("title").Default(""),
		field.JSON("data", map[string]any{}).Optional(),
		field.String("created_by").Default(""),
		field.Time("expires_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes of the PageData.
func (PageData) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token").Unique(),
	}
}

// Annotations of the PageData.
func (PageData) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("page_data"),
	}
}

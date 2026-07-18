package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Clip stores a shareable markdown clip addressed by a short slug.
type Clip struct {
	ent.Schema
}

// Fields of the Clip.
func (Clip) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("slug").NotEmpty().Unique(),
		field.String("title").Default(""),
		field.String("description").Default(""),
		field.Text("content").NotEmpty(),
		field.String("created_by").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Annotations of the Clip.
func (Clip) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("clips"),
	}
}

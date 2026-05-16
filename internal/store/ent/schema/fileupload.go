package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Fileupload struct {
	ent.Schema
}

func (Fileupload) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("fid").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("mimetype").NotEmpty(),
		field.Int64("size").Default(0),
		field.String("location").NotEmpty(),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Fileupload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("state"),
		index.Fields("uid"),
		index.Fields("fid"),
	}
}

func (Fileupload) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("fileuploads"),
	}
}

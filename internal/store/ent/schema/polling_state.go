package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type PollingState struct {
	ent.Schema
}

func (PollingState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("resource_name").NotEmpty().Unique(),
		field.Text("cursor").NotEmpty().Default(""),
		field.JSON("known_hashes", map[string]string{}).Default(map[string]string{}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PollingState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("polling_state"),
	}
}

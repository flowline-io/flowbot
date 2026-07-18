package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// NotifyTemplateOverride is a channel-specific template body stored as JSON.
type NotifyTemplateOverride struct {
	Channel  string `json:"channel"`
	Format   string `json:"format"`
	Template string `json:"template"`
}

// NotifyTemplate holds a notification message template persisted in the database.
type NotifyTemplate struct {
	ent.Schema
}

// Fields of the NotifyTemplate.
func (NotifyTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("template_id").Unique().NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("description").Optional().Default(""),
		field.String("default_format").Default("markdown").NotEmpty(),
		field.Text("default_template").NotEmpty(),
		field.JSON("overrides", []NotifyTemplateOverride{}).Default([]NotifyTemplateOverride{}),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the NotifyTemplate.
func (NotifyTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("template_id"),
	}
}

// Annotations of the NotifyTemplate.
func (NotifyTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notify_templates"),
	}
}

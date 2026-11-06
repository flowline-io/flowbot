package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/flowline-io/flowbot/pkg/types"
)

type ResourceLink struct {
	ent.Schema
}

func (ResourceLink) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("source_event_id").NotEmpty(),
		field.String("target_event_id").NotEmpty(),
		field.String("source_app").Default(""),
		field.String("target_app").Default(""),
		field.String("source_capability").Default(""),
		field.String("target_capability").Default(""),
		field.String("source_entity_id").Default(""),
		field.String("target_entity_id").Default(""),
		field.Int64("pipeline_run_id").Optional(),
		field.String("pipeline_name").Default(""),
		field.Enum("run_kind").
			Values(
				string(types.ResourceLinkRunKindPipeline),
				string(types.ResourceLinkRunKindWorkflow),
				string(types.ResourceLinkRunKindFunction),
				string(types.ResourceLinkRunKindMCP),
				string(types.ResourceLinkRunKindManual),
			).
			Default(string(types.ResourceLinkRunKindPipeline)),
		field.Enum("status").
			Values(
				string(types.ResourceLinkStatusOK),
				string(types.ResourceLinkStatusFailed),
			).
			Default(string(types.ResourceLinkStatusOK)),
		field.String("correlation_id").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (ResourceLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_event_id", "target_event_id").Unique(),
		index.Fields("source_app", "source_entity_id"),
		index.Fields("target_app", "target_entity_id"),
		index.Fields("source_event_id"),
		index.Fields("target_event_id"),
		index.Fields("source_app", "target_app", "created_at"),
		index.Fields("correlation_id"),
	}
}

func (ResourceLink) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("resource_links"),
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Dag struct {
	ent.Schema
}

func (Dag) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id").Optional(),
		field.Int64("script_id"),
		field.Int32("script_version").Default(0),
		field.JSON("nodes", []map[string]any{}).StorageKey("nodes"),
		field.JSON("edges_data", []map[string]any{}).StorageKey("edges"),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
	}
}

func (Dag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).Ref("dag").Unique().Field("workflow_id"),
	}
}

func (Dag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_id"),
	}
}

func (Dag) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("dag"),
	}
}

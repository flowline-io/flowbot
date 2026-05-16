package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type FlowEdge struct {
	ent.Schema
}

func (FlowEdge) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("flow_id"),
		field.String("edge_id").NotEmpty(),
		field.String("source_node").NotEmpty(),
		field.String("target_node").NotEmpty(),
		field.String("source_port").Optional().Default(""),
		field.String("target_port").Optional().Default(""),
		field.String("label").Optional().Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FlowEdge) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("flow_edges"),
	}
}

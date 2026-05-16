package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
)

type FlowNode struct {
	ent.Schema
}

func (FlowNode) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("flow_id"),
		field.String("node_id").NotEmpty(),
		field.String("type").NotEmpty(),
		field.String("bot").NotEmpty(),
		field.String("rule_id").NotEmpty(),
		field.String("label").Optional().Default(""),
		field.Int("position_x").Default(0),
		field.Int("position_y").Default(0),
		field.JSON("parameters", map[string]any{}).Optional(),
		field.JSON("variables", map[string]any{}).Optional(),
		field.JSON("conditions", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FlowNode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("flow_nodes"),
	}
}

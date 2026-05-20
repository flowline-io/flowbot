package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type WorkflowTrigger struct {
	ent.Schema
}

func (WorkflowTrigger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id").Optional(),
		field.String("type").NotEmpty(),
		field.JSON("rule", map[string]any{}).Optional(),
		field.Int32("count").Default(0),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WorkflowTrigger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).Ref("triggers").Unique().Field("workflow_id"),
	}
}

func (WorkflowTrigger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_trigger"),
	}
}

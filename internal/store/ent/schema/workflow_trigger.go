package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkflowTrigger stores one trigger belonging to a workflow definition.
type WorkflowTrigger struct {
	ent.Schema
}

// Fields of the WorkflowTrigger.
func (WorkflowTrigger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("workflow_id"),
		field.String("type").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.JSON("rule", map[string]any{}).Optional(),
	}
}

// Indexes of the WorkflowTrigger.
func (WorkflowTrigger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_id"),
	}
}

// Annotations of the WorkflowTrigger.
func (WorkflowTrigger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow_triggers"),
	}
}

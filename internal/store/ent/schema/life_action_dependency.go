package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifeActionDependency stores one checkpoint-to-todo dependency edge.
type LifeActionDependency struct {
	ent.Schema
}

// Fields of the LifeActionDependency.
func (LifeActionDependency) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.Int64("action_plan_node_id"),
		field.Int64("depends_on_plan_node_id"),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Indexes of the LifeActionDependency.
func (LifeActionDependency) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("action_plan_node_id", "depends_on_plan_node_id").Unique(),
		index.Fields("depends_on_plan_node_id"),
	}
}

// Annotations of the LifeActionDependency.
func (LifeActionDependency) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_action_dependencies"),
	}
}

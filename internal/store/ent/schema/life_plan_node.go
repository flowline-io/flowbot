package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LifePlanNode is one node in the life planning tree.
type LifePlanNode struct {
	ent.Schema
}

// Fields of the LifePlanNode.
func (LifePlanNode) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("flag").NotEmpty().Unique(),
		field.Int64("life_profile_id"),
		field.Int64("parent_id").Optional().Nillable(),
		field.String("node_type").NotEmpty(),
		field.String("title").NotEmpty(),
		field.Text("description").Default(""),
		field.String("status").Default("Active"),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the LifePlanNode.
func (LifePlanNode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("life_profile_id"),
		index.Fields("life_profile_id", "node_type"),
		index.Fields("life_profile_id", "parent_id", "sort_order"),
	}
}

// Annotations of the LifePlanNode.
func (LifePlanNode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("life_plan_nodes"),
	}
}

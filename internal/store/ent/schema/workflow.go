package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Workflow struct {
	ent.Schema
}

func (Workflow) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.String("flag").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("describe").Default(""),
		field.Int32("successful_count").Default(0),
		field.Int32("failed_count").Default(0),
		field.Int32("running_count").Default(0),
		field.Int32("canceled_count").Default(0),
		field.Int("state").Default(0),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Workflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("dag", Dag.Type).Unique(),
		edge.To("triggers", WorkflowTrigger.Type),
	}
}

func (Workflow) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("workflow"),
	}
}

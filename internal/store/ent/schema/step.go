package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Step struct {
	ent.Schema
}

func (Step) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("topic").NotEmpty(),
		field.Int64("job_id").Optional(),
		field.JSON("action", map[string]any{}),
		field.String("name").Default(""),
		field.String("describe").Default(""),
		field.String("node_id").Default(""),
		field.JSON("depend", []string{}),
		field.JSON("input", map[string]any{}).Optional(),
		field.JSON("output", map[string]any{}).Optional(),
		field.String("error").Optional().Default(""),
		field.Int("state").Default(0),
		field.Time("started_at").Optional().Nillable(),
		field.Time("ended_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Step) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("job", Job.Type).Ref("steps").Unique().Field("job_id"),
	}
}

func (Step) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "topic"),
		index.Fields("job_id"),
		index.Fields("node_id"),
		index.Fields("state"),
	}
}

func (Step) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("steps"),
	}
}

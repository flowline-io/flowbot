package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuditLog struct {
	ent.Schema
}

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("action").NotEmpty(),
		field.String("target_type").NotEmpty(),
		field.String("target_id").NotEmpty(),
		field.String("actor_uid").Default(""),
		field.String("result").Default(""),
		field.String("error_code").Default(""),
		field.JSON("details", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("actor_uid", "created_at"),
		index.Fields("action", "created_at"),
		index.Fields("result", "created_at"),
	}
}

func (AuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("audit_logs"),
	}
}

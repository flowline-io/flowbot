package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
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
		field.JSON("details", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (AuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("audit_logs"),
	}
}

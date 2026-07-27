package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WebAccount stores web UI login credentials and TOTP state.
type WebAccount struct {
	ent.Schema
}

// Fields of the WebAccount.
func (WebAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("username").NotEmpty(),
		field.String("uid").NotEmpty(),
		field.String("password_hash").NotEmpty(),
		field.Bytes("totp_secret_ciphertext").Optional().Nillable(),
		field.Bytes("totp_secret_nonce").Optional().Nillable(),
		field.Bool("totp_enabled").Default(false),
		field.Int64("totp_last_step").Default(0),
		field.JSON("backup_code_hashes", []string{}).Default([]string{}),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the WebAccount.
func (WebAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique(),
		index.Fields("uid").Unique(),
	}
}

// Annotations of the WebAccount.
func (WebAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("web_accounts"),
	}
}

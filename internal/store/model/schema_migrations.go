package model

// SchemaMigration mapped from table <schema_migrations>
type SchemaMigration struct {
	Version int64 `json:"version"`
	Dirty   bool  `json:"dirty"`
}

package model

import "time"

// FunctionDefinition is a named function definition row for catalog and UI use.
type FunctionDefinition struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Status              string    `json:"status"`
	Version             int       `json:"version,omitempty"`
	CreatedBy           string    `json:"created_by,omitempty"`
	MetadataDraft       string    `json:"metadata_draft,omitempty"`
	EntrypointDraft     string    `json:"entrypoint_draft,omitempty"`
	SourceDraft         string    `json:"source_draft,omitempty"`
	MetadataPublished   *string   `json:"metadata_published,omitempty"`
	EntrypointPublished *string   `json:"entrypoint_published,omitempty"`
	SourcePublished     *string   `json:"source_published,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// FunctionDefinitionVersion is an immutable published snapshot of a named function.
type FunctionDefinitionVersion struct {
	ID           int64     `json:"id"`
	FunctionName string    `json:"function_name"`
	Version      int       `json:"version"`
	Metadata     string    `json:"metadata"`
	Entrypoint   string    `json:"entrypoint"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
}

// FunctionRun is one invocation of a named function for audit and idempotent replay.
type FunctionRun struct {
	ID             int64     `json:"id"`
	FunctionName   string    `json:"function_name"`
	Version        int       `json:"version"`
	Status         string    `json:"status"`
	DurationMs     int64     `json:"duration_ms,omitempty"`
	ExitCode       *int      `json:"exit_code,omitempty"`
	Error          string    `json:"error,omitempty"`
	ResultJSON     *string   `json:"result_json,omitempty"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

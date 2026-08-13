package types

// FunctionRunState is the lifecycle state of a named function run.
type FunctionRunState string

const (
	// FunctionRunRunning means the invocation has been recorded and is in progress.
	FunctionRunRunning FunctionRunState = "running"
	// FunctionRunSucceeded means the invocation finished with a parsed JSON result.
	FunctionRunSucceeded FunctionRunState = "succeeded"
	// FunctionRunFailed means the invocation failed (non-zero exit, bad JSON, or runtime error).
	FunctionRunFailed FunctionRunState = "failed"
)

// FunctionDefinitionStatus is the draft/published status of a named function definition.
type FunctionDefinitionStatus string

const (
	// FunctionDefinitionDraft means the definition has draft fields not yet published (or never published).
	FunctionDefinitionDraft FunctionDefinitionStatus = "draft"
	// FunctionDefinitionPublished means at least one immutable version snapshot exists.
	FunctionDefinitionPublished FunctionDefinitionStatus = "published"
)

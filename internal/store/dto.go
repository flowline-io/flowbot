// Package store provides database storage implementations.
package store

import (
	"time"
)

// ListConfigOptions controls pagination and search for ListConfigs.
type ListConfigOptions struct {
	Offset int
	Limit  int
	Search string
}

// ListNotifyChannelOptions holds filtering options for listing notification channels.
type ListNotifyChannelOptions struct {
	Protocol string
	Enabled  *bool // nil = all, true = enabled only, false = disabled only
}

// ListNotifyRuleOptions holds filtering and sorting options for listing notification rules.
type ListNotifyRuleOptions struct {
	Enabled *bool // nil = all, true = enabled only, false = disabled only
}

// ListNotifyTemplateOptions holds filtering options for listing notification templates.
type ListNotifyTemplateOptions struct{}

// AgentKnowledgeListFilter filters knowledge documents for the management UI.
type AgentKnowledgeListFilter struct {
	// Q matches a substring of path or title when non-empty.
	Q string
}

// AgentKnowledgeSearchParams controls agent knowledge search over path, title,
// tags, summary, content (DB substring match), plus optional exact tag filter.
type AgentKnowledgeSearchParams struct {
	// Query matches path, title, tags, summary, or content via case-insensitive substring.
	Query string
	// PathPrefix restricts results to paths with this prefix.
	PathPrefix string
	// Tag requires an exact tag match when non-empty.
	Tag string
	// Limit caps results (default 10, max 50).
	Limit int
}

// AgentMemoryFactUpsert carries fields for inserting or updating one memory fact.
type AgentMemoryFactUpsert struct {
	// Scope isolates facts (interactive chat uses "default").
	Scope string
	// Key is the fact name within the scope.
	Key string
	// Value is the stored fact text.
	Value string
	// Pinned prefers the fact for system-prompt injection.
	Pinned bool
}

// AgentMemoryInjectableParams controls which facts are selected for system-prompt injection.
type AgentMemoryInjectableParams struct {
	// Scope selects which memory scope to read.
	Scope string
	// MaxCount caps how many facts are returned (default 30).
	MaxCount int
	// MaxChars caps total key+value characters across returned facts (default 4000).
	MaxChars int
}

// AgentMemoryFactsFingerprint is a cache-busting digest for injectable facts in a scope.
type AgentMemoryFactsFingerprint struct {
	// Count is the number of facts in the scope.
	Count int
	// MaxUpdatedAt is the newest updated_at among facts in the scope.
	MaxUpdatedAt time.Time
	// ContentHash digests key/value/pinned/updated_at for cache invalidation.
	ContentHash string
}

// AgentSessionSummarySearchParams controls session summary search.
type AgentSessionSummarySearchParams struct {
	// Query matches title or summary via case-insensitive substring.
	Query string
	// Scope restricts results when non-empty.
	Scope string
	// Limit caps results (default 10, max 50).
	Limit int
}

// AgentSessionSummaryListFilter filters session summaries for the management UI.
type AgentSessionSummaryListFilter struct {
	// Scope restricts results when non-empty.
	Scope string
	// Status filters by pending|ready|failed when non-empty.
	Status string
	// Q matches title or summary via case-insensitive substring when non-empty.
	Q string
}

// ListChatSessionsOptions holds pagination for listing chat agent sessions.
type ListChatSessionsOptions struct {
	Limit  int    // max 100, default 20
	Cursor string // opaque cursor: session ID value as string
	UID    string // when set, only sessions owned by this user are returned
	State  *int   // when set, only sessions in this state are returned
	// Archived filters by archive flag; nil means any.
	Archived *bool
	// PinnedFirst sorts pinned sessions ahead of unpinned ones.
	PinnedFirst bool
	// Flags restricts results to the given session flags when non-empty.
	Flags []string
}

// ListChatScheduledTasksOptions filters scheduled task queries.
type ListChatScheduledTasksOptions struct {
	UID    string
	States []string
}

// UpdateChatScheduledTaskParams carries partial updates for a scheduled task row.
type UpdateChatScheduledTaskParams struct {
	Name      *string
	Cron      *string
	RunAt     *time.Time
	Prompt    *string
	State     *string
	LastRunAt *time.Time
	NextRunAt *time.Time
}

// UpdateChatScheduledTaskRunParams carries partial updates for one run row.
type UpdateChatScheduledTaskRunParams struct {
	State      *string
	Reply      *string
	Error      *string
	FinishedAt *time.Time
}

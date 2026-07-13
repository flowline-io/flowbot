package capability

// BookmarkListQuery wraps pagination and filters for listing bookmarks.
type BookmarkListQuery struct {
	Page       PageRequest
	Archived   *bool
	Favourited *bool
	Tags       []string
}

// BookmarkSearchQuery wraps pagination for searching bookmarks.
type BookmarkSearchQuery struct {
	Page PageRequest
	Q    string
}

// ReaderFeedQuery wraps pagination for listing feeds.
type ReaderFeedQuery struct {
	Page PageRequest
}

// ReaderEntryQuery wraps pagination and filters for listing entries.
type ReaderEntryQuery struct {
	Page   PageRequest
	Status string
	FeedID int64
}

// KanbanTaskQuery wraps pagination and filters for listing tasks.
type KanbanTaskQuery struct {
	Page      PageRequest
	ProjectID int
	ColumnID  int
	Status    string
}

// KanbanCreateTaskRequest holds fields for creating a task.
type KanbanCreateTaskRequest struct {
	Title       string
	Description string
	ProjectID   int
	ColumnID    int
	Tags        []string
	Reference   string
}

// KanbanUpdateTaskRequest holds fields for updating a task.
type KanbanUpdateTaskRequest struct {
	Title       string
	Description string
}

// KanbanMoveTaskRequest holds fields for moving a task.
type KanbanMoveTaskRequest struct {
	ColumnID   int
	Position   int
	SwimlaneID int
	ProjectID  int
}

// KanbanSearchQuery wraps pagination for searching tasks.
type KanbanSearchQuery struct {
	Page      PageRequest
	ProjectID int
	Q         string
}

// NoteListQuery wraps pagination for listing notes.
type NoteListQuery struct {
	Page  PageRequest
	Query string
}

// MemoListQuery wraps pagination for listing memos.
type MemoListQuery struct {
	Page PageRequest
}

// ForgeListIssuesQuery wraps pagination and filtering for listing forge issues.
type ForgeListIssuesQuery struct {
	Page  PageRequest
	State string
}

// GithubListIssuesQuery wraps pagination and filtering for listing GitHub issues.
type GithubListIssuesQuery struct {
	Page  PageRequest
	State string
}

// GithubPageQuery wraps pagination for GitHub list operations.
type GithubPageQuery struct {
	Page PageRequest
}

// ExampleListQuery wraps pagination for listing example items.
type ExampleListQuery struct {
	Page PageRequest
}

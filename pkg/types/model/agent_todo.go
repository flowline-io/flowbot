package model

// AgentTodo represents one session checklist item for UI display.
type AgentTodo struct {
	ItemID    string `json:"item_id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	SortOrder int    `json:"sort_order"`
}

// AgentTodoSummary is aggregate checklist progress for session list display.
type AgentTodoSummary struct {
	Total      int    `json:"total"`
	Done       int    `json:"done"`
	Active     int    `json:"active"`
	InProgress string `json:"in_progress,omitempty"`
}

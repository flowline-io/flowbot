// Package example implements the example provider using jsonplaceholder.typicode.com for demonstration.
package example

// MaxPageSize is the maximum number of items per page.
const MaxPageSize = 100

// Response mirrors jsonplaceholder post response JSON structure.
type Response struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// WebhookPayload represents a webhook event payload from the example provider.
type WebhookPayload struct {
	EventType string `json:"event_type"`
	EntityID  string `json:"entity_id"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

// ItemsResponse carries a list of items with cursor for polling.
type ItemsResponse struct {
	Items      []map[string]any `json:"items"`
	NextCursor string           `json:"next_cursor"`
}

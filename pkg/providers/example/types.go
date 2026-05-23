// Package example implements the example provider using httpbin.org for demonstration.
package example

// MaxPageSize is the maximum number of items per page.
const MaxPageSize = 100

// Response mirrors httpbin response JSON structure for GET/POST/PUT/DELETE endpoints.
type Response struct {
	Args    map[string]string `json:"args"`
	Data    string            `json:"data"`
	Files   map[string]string `json:"files"`
	Form    map[string]string `json:"form"`
	Headers map[string]string `json:"headers"`
	JSON    any               `json:"json"`
	Method  string            `json:"method"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
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

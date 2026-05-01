package ability

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/hub"
)

type Bookmark struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Title      string    `json:"title,omitzero"`
	Summary    string    `json:"summary,omitzero"`
	Tags       []string  `json:"tags,omitzero"`
	Archived   bool      `json:"archived"`
	Favourited bool      `json:"favourited"`
	CreatedAt  time.Time `json:"created_at"`
}

type ArchiveItem struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title,omitzero"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Feed struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	FeedURL  string `json:"feed_url"`
	SiteURL  string `json:"site_url,omitzero"`
	Category string `json:"category,omitzero"`
}

type Entry struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content,omitzero"`
	Status      string    `json:"status"`
	Starred     bool      `json:"starred"`
	PublishedAt time.Time `json:"published_at"`
	FeedTitle   string    `json:"feed_title,omitzero"`
}

type Task struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitzero"`
	ProjectID   int      `json:"project_id"`
	ColumnID    int      `json:"column_id"`
	Tags        []string `json:"tags,omitzero"`
	Reference   string   `json:"reference,omitzero"`
}

type Host struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitzero"`
	Status  string `json:"status"`
}

type InvokeResult struct {
	Capability hub.CapabilityType `json:"capability"`
	Operation  string             `json:"operation"`
	Data       any                `json:"data,omitzero"`
	Page       *PageInfo          `json:"page,omitzero"`
	Text       string             `json:"text,omitzero"`
	Meta       map[string]any     `json:"meta,omitzero"`
	Events     []EventRef         `json:"events,omitzero"`
}

type EventRef struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	EntityID  string `json:"entity_id,omitzero"`
}

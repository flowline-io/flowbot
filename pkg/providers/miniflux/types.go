package miniflux

import "time"

const (
	NewEntriesEventType = "new_entries"
	SaveEntryEventType  = "save_entry"
)

type WebhookFeed struct {
	ID         int64            `json:"id"`
	UserID     int64            `json:"user_id"`
	CategoryID int64            `json:"category_id"`
	Category   *WebhookCategory `json:"category,omitempty"`
	FeedURL    string           `json:"feed_url"`
	SiteURL    string           `json:"site_url"`
	Title      string           `json:"title"`
	CheckedAt  time.Time        `json:"checked_at"`
}

type WebhookCategory struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type WebhookEntry struct {
	ID          int64        `json:"id"`
	UserID      int64        `json:"user_id"`
	FeedID      int64        `json:"feed_id"`
	Status      string       `json:"status"`
	Hash        string       `json:"hash"`
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	CommentsURL string       `json:"comments_url"`
	Date        time.Time    `json:"published_at"`
	CreatedAt   time.Time    `json:"created_at"`
	ChangedAt   time.Time    `json:"changed_at"`
	Content     string       `json:"content"`
	Author      string       `json:"author"`
	ShareCode   string       `json:"share_code"`
	Starred     bool         `json:"starred"`
	ReadingTime int          `json:"reading_time"`
	Tags        []string     `json:"tags"`
	Feed        *WebhookFeed `json:"feed,omitempty"`
	//Enclosures  model.EnclosureList `json:"enclosures"`
}

type WebhookEvent struct {
	EventType string          `json:"event_type"`
	Feed      *WebhookFeed    `json:"feed"`
	Entries   []*WebhookEntry `json:"entries"`
	Entry     *WebhookEntry   `json:"entry"`
}

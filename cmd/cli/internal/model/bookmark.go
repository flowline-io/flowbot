package model

import "time"

// Bookmark represents a saved URL bookmark
type Bookmark struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BookmarkStore holds all bookmarks
type BookmarkStore struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

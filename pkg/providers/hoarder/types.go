package hoarder

import "time"

const (
	MaxPageSize = 100
)

type BookmarksResponse struct {
	Bookmarks  []Bookmark `json:"bookmarks"`
	NextCursor string     `json:"nextCursor"`
}

type Bookmark struct {
	Id            string      `json:"id"`
	CreatedAt     time.Time   `json:"createdAt"`
	Title         string      `json:"title"`
	Archived      bool        `json:"archived"`
	Favourited    bool        `json:"favourited"`
	TaggingStatus string      `json:"taggingStatus"`
	Note          interface{} `json:"note"`
	Summary       interface{} `json:"summary"`
	Tags          []Tag       `json:"tags"`
	Content       struct {
		Type        string      `json:"type"`
		Url         string      `json:"url"`
		Title       interface{} `json:"title"`
		Description interface{} `json:"description"`
		ImageUrl    interface{} `json:"imageUrl"`
		Favicon     interface{} `json:"favicon"`
		HtmlContent string      `json:"htmlContent"`
		CrawledAt   interface{} `json:"crawledAt"`
	} `json:"content"`
	Assets []interface{} `json:"assets"`
}

type Tag struct {
	Id                         string `json:"id"`
	Name                       string `json:"name"`
	AttachedBy                 string `json:"attachedBy"`
	NumBookmarks               int    `json:"numBookmarks"`
	NumBookmarksByAttachedType struct {
		Ai    int `json:"ai"`
		Human int `json:"human"`
	} `json:"numBookmarksByAttachedType"`
}

type TagsResponse struct {
	Tags []Tag `json:"tags"`
}

type AttachedResponse struct {
	Attached []string `json:"attached"`
}

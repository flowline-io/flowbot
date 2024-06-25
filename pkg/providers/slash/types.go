package slash

import "time"

type Shortcut struct {
	Id          int32     `json:"id,omitempty"`
	CreatorId   int32     `json:"creator_id,omitempty"`
	CreatedTime time.Time `json:"created_time"`
	UpdatedTime time.Time `json:"updated_time"`
	RowStatus   any       `json:"row_status,omitempty"`
	Name        string    `json:"name,omitempty"`
	Link        string    `json:"link,omitempty"`
	Title       string    `json:"title,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Description string    `json:"description,omitempty"`
	Visibility  any       `json:"visibility,omitempty"`
	ViewCount   int32     `json:"view_count,omitempty"`
	OGMetaData  struct {
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		Image       string `json:"image,omitempty"`
	} `json:"og_metadata"`
}

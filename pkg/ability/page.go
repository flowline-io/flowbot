package ability

type PageRequest struct {
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

type PageInfo struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	Total      *int64 `json:"total,omitempty"`
}

type ListResult[T any] struct {
	Items []*T      `json:"items"`
	Page  *PageInfo `json:"page,omitempty"`
}

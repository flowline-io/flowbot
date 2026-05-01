package ability

type PageRequest struct {
	Limit     int    `json:"limit,omitzero"`
	Cursor    string `json:"cursor,omitzero"`
	SortBy    string `json:"sort_by,omitzero"`
	SortOrder string `json:"sort_order,omitzero"`
}

type PageInfo struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
	PrevCursor string `json:"prev_cursor,omitzero"`
	Total      *int64 `json:"total,omitzero"`
}

type ListResult[T any] struct {
	Items []*T      `json:"items"`
	Page  *PageInfo `json:"page,omitzero"`
}

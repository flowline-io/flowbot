package capability

import "time"

// PaginateOffsetSlice pages an in-memory slice with opaque signed cursors.
func PaginateOffsetSlice[T any](items []*T, pageReq PageRequest, secret []byte, now time.Time, capName string) *ListResult[T] {
	offset := 0
	limit := normalizedOffsetLimit(pageReq.Limit)
	if pageReq.Cursor != "" {
		if payload, err := DecodeCursor(secret, pageReq.Cursor, now); err == nil {
			offset = payload.Offset
			if payload.Limit > 0 {
				limit = payload.Limit
			}
		}
	}
	if offset >= len(items) {
		return &ListResult[T]{Items: []*T{}, Page: &PageInfo{Limit: limit}}
	}
	end := min(offset+limit, len(items))
	slice := items[offset:end]
	hasMore := end < len(items)
	page := &PageInfo{Limit: limit, HasMore: hasMore}
	if hasMore {
		nextCursor, err := EncodeCursor(secret, CursorPayload{
			Capability: capName,
			Strategy:   "offset",
			Offset:     end,
			Limit:      limit,
		})
		if err == nil {
			page.NextCursor = nextCursor
		}
	}
	return &ListResult[T]{Items: slice, Page: page}
}

// OffsetPageFromCursor decodes an opaque cursor into a start offset and limit.
func OffsetPageFromCursor(cursor string, limit int, secret []byte, now time.Time, capName string) (start, pageLimit int) {
	pageLimit = normalizedOffsetLimit(limit)
	start = 0
	if cursor == "" {
		return start, pageLimit
	}
	payload, err := DecodeCursor(secret, cursor, now)
	if err != nil || payload.Capability != capName {
		return start, pageLimit
	}
	start = payload.Offset
	if payload.Limit > 0 {
		pageLimit = payload.Limit
	}
	return start, pageLimit
}

// OffsetPageInfo builds opaque next cursor from provider start/limit/size.
func OffsetPageInfo(start, limit, size, resultCount int, secret []byte, capName string) *PageInfo {
	hasMore := start+resultCount < size
	page := &PageInfo{Limit: limit, HasMore: hasMore}
	if hasMore {
		nextStart := start + resultCount
		nextCursor, err := EncodeCursor(secret, CursorPayload{
			Capability: capName,
			Strategy:   "offset",
			Offset:     nextStart,
			Limit:      limit,
		})
		if err == nil {
			page.NextCursor = nextCursor
		}
	}
	return page
}

func normalizedOffsetLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

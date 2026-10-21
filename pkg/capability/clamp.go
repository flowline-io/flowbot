package capability

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/bytedance/sonic"
)

// fullListSecret signs cursors for lists that return every row and have no provider page.
var fullListSecret = []byte("flowbot-capability-full-list-cursor-v1")

// ApplyFullListPage slices data when the handler returned every row without a page.
// Lists that already set limit, has_more, next_cursor, or total are left unchanged.
func ApplyFullListPage(ctx context.Context, capName string, params map[string]any, data any, page *PageInfo) (any, *PageInfo) {
	if pageReady(page) {
		return data, page
	}
	req := PageRequest{}
	if params != nil {
		req = PageRequestFromParams(params)
	}
	sliced, next := pageFullSlice(data, req, capName)
	if next == nil {
		return data, page
	}
	PublishCursorSecret(ctx, fullListSecret)
	return sliced, next
}

// ClampListed shortens a list result when ctx carries an output budget.
// secret is the HMAC key for an existing cursor. A secret published on ctx wins.
func ClampListed(ctx context.Context, capName string, params map[string]any, data any, page *PageInfo, secret []byte) (any, *PageInfo) {
	if secret != nil {
		PublishCursorSecret(ctx, secret)
	}
	if published := publishedCursorSecret(ctx); len(published) > 0 {
		secret = published
	}
	req := PageRequest{}
	if params != nil {
		req = PageRequestFromParams(params)
	}
	return sealItems(ctx, data, page, secret, req, capName)
}

func fullListLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}

func pageReady(page *PageInfo) bool {
	if page == nil {
		return false
	}
	return page.Limit > 0 || page.HasMore || page.NextCursor != "" || page.Total != nil
}

func pageFullSlice(data any, req PageRequest, capName string) (any, *PageInfo) {
	rv := reflect.ValueOf(data)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return data, nil
	}
	items := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		items[i] = rv.Index(i).Interface()
	}
	limit := fullListLimit(req.Limit)
	start := 0
	if req.Cursor != "" {
		payload, err := DecodeCursor(fullListSecret, req.Cursor, time.Now())
		if err == nil && payload.Strategy == "offset" && payload.ProviderCursor == "" {
			start = payload.Offset
			if payload.Limit > 0 {
				limit = payload.Limit
			}
		}
	}
	if start > len(items) {
		start = len(items)
	}
	end := min(start+limit, len(items))
	page := &PageInfo{Limit: limit, HasMore: end < len(items)}
	if page.HasMore {
		cursor, err := EncodeCursor(fullListSecret, CursorPayload{
			Capability: capName,
			Strategy:   "offset",
			Offset:     end,
			Limit:      limit,
		})
		if err == nil {
			page.NextCursor = cursor
		}
	}
	return items[start:end], page
}

func sealItems(ctx context.Context, data any, page *PageInfo, secret []byte, req PageRequest, capName string) (any, *PageInfo) {
	budget, ok := pageBudgetFrom(ctx)
	if !ok {
		return data, page
	}
	items := asAnySlice(data)
	if items == nil && data != nil {
		return data, page
	}
	if items == nil {
		items = []any{}
	}
	generic, err := roundTripSlice(items)
	if err != nil {
		return data, page
	}
	if page == nil {
		page = &PageInfo{}
	}
	sealed := *page
	addressable, boundary := itemBoundary(generic, &sealed, req, secret)
	generic, sealed, changed := fitWithinBudget(budget, generic, sealed, addressable, boundary, secret, capName)
	if !changed {
		return data, page
	}
	return generic, &sealed
}

func fitWithinBudget(budget PageBudget, generic []any, sealed PageInfo, addressable bool, boundary int, secret []byte, capName string) ([]any, PageInfo, bool) {
	originalNext := sealed.NextCursor
	dropped := 0
	changed := false
	for range len(generic)*4 + 8 {
		if !pageOverBudget(budget, generic, &sealed) {
			break
		}
		changed = true
		sealed.Truncated = true
		if dropAddressableTail(&generic, &sealed, addressable, &dropped, boundary, secret, capName) {
			continue
		}
		if shrinkLongest(generic) {
			continue
		}
		if !stubTail(&generic, &sealed, addressable, originalNext, budget, &dropped, boundary, secret, capName) {
			break
		}
	}
	return generic, sealed, changed
}

func pageOverBudget(budget PageBudget, items []any, page *PageInfo) bool {
	size, err := budget.Measure(items, pageToMap(page))
	return err == nil && size > budget.Max
}

func dropAddressableTail(items *[]any, page *PageInfo, addressable bool, dropped *int, boundary int, secret []byte, capName string) bool {
	if !addressable || len(*items) <= 1 {
		return false
	}
	*items = (*items)[:len(*items)-1]
	*dropped++
	writeOffsetCursor(page, secret, capName, boundary-*dropped, page.Limit)
	return true
}

func stubTail(items *[]any, page *PageInfo, addressable bool, originalNext string, budget PageBudget, dropped *int, boundary int, secret []byte, capName string) bool {
	if len(*items) == 0 {
		return false
	}
	if !addressable {
		idx := lastUnstubbed(*items)
		if idx < 0 {
			return false
		}
		(*items)[idx] = stubItem((*items)[idx])
		page.NextCursor = originalNext
		page.HasMore = originalNext != "" || page.HasMore
		return pageOverBudget(budget, *items, page)
	}
	last := len(*items) - 1
	(*items)[last] = stubItem((*items)[last])
	if !pageOverBudget(budget, *items, page) || len(*items) == 1 {
		return false
	}
	*items = (*items)[:last]
	*dropped++
	writeOffsetCursor(page, secret, capName, boundary-*dropped, page.Limit)
	return true
}

func lastUnstubbed(items []any) int {
	for i, item := range slices.Backward(items) {
		if !isStub(item) {
			return i
		}
	}
	return -1
}

func isStub(item any) bool {
	row, ok := item.(map[string]any)
	if !ok {
		return false
	}
	_, ok = row["_truncated"]
	return ok
}

func itemBoundary(items []any, page *PageInfo, req PageRequest, secret []byte) (addressable bool, boundary int) {
	now := time.Now()
	if page != nil && page.NextCursor != "" {
		payload, ok := decodeItemOffset(secret, page.NextCursor, now)
		if !ok {
			return false, 0
		}
		return true, payload.Offset
	}
	if req.Cursor != "" {
		payload, ok := decodeItemOffset(secret, req.Cursor, now)
		if !ok {
			return false, 0
		}
		return true, payload.Offset + len(items)
	}
	return true, len(items)
}

func decodeItemOffset(secret []byte, cursor string, now time.Time) (CursorPayload, bool) {
	if cursor == "" || len(secret) == 0 {
		return CursorPayload{}, false
	}
	payload, err := DecodeCursor(secret, cursor, now)
	if err != nil || payload.Strategy != "offset" || payload.ProviderCursor != "" {
		return CursorPayload{}, false
	}
	return payload, true
}

func writeOffsetCursor(page *PageInfo, secret []byte, capName string, offset, limit int) {
	if page == nil {
		return
	}
	if limit <= 0 {
		limit = normalizedOffsetLimit(0)
	}
	page.Limit = limit
	if offset < 0 {
		offset = 0
	}
	page.HasMore = true
	page.Truncated = true
	if len(secret) == 0 {
		page.NextCursor = ""
		return
	}
	cursor, err := EncodeCursor(secret, CursorPayload{
		Capability: capName,
		Strategy:   "offset",
		Offset:     offset,
		Limit:      limit,
	})
	if err != nil {
		page.NextCursor = ""
		return
	}
	page.NextCursor = cursor
}

func asAnySlice(data any) []any {
	if data == nil {
		return []any{}
	}
	if items, ok := data.([]any); ok {
		return items
	}
	rv := reflect.ValueOf(data)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil
	}
	items := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		items[i] = rv.Index(i).Interface()
	}
	return items
}

func roundTripSlice(items []any) ([]any, error) {
	raw, err := sonic.Marshal(items)
	if err != nil {
		return nil, err
	}
	var out []any
	if err := sonic.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []any{}
	}
	return out, nil
}

func pageToMap(page *PageInfo) map[string]any {
	if page == nil {
		return map[string]any{}
	}
	raw, err := sonic.Marshal(page)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := sonic.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func stubItem(item any) map[string]any {
	raw, err := sonic.Marshal(item)
	n := 0
	if err == nil {
		n = len(raw)
	}
	return map[string]any{
		"_truncated": StubItemMessage,
		"bytes":      n,
	}
}

func stringAt(parent any, key string, idx int) (string, bool) {
	switch p := parent.(type) {
	case map[string]any:
		value, ok := p[key].(string)
		return value, ok
	case []any:
		if idx < 0 || idx >= len(p) {
			return "", false
		}
		value, ok := p[idx].(string)
		return value, ok
	default:
		return "", false
	}
}

func shrinkLongest(items []any) bool {
	parent, key, idx, ok := findLongestString(items)
	if !ok {
		return false
	}
	current, ok := stringAt(parent, key, idx)
	if !ok {
		return false
	}
	next, shrunk := truncateString(current)
	if !shrunk {
		return false
	}
	switch p := parent.(type) {
	case map[string]any:
		p[key] = next
	case []any:
		p[idx] = next
	}
	return true
}

func truncateString(s string) (string, bool) {
	if len(s) < 2 {
		return s, false
	}
	removed := max(len(s)/2, 1)
	marker := fmt.Sprintf("…[truncated %d bytes]", removed)
	for len(s)-removed+len(marker) >= len(s) && removed < len(s) {
		removed++
		marker = fmt.Sprintf("…[truncated %d bytes]", removed)
	}
	if removed >= len(s) {
		marker = fmt.Sprintf("…[truncated %d bytes]", len(s))
		if len(marker) >= len(s) {
			return s, false
		}
		return marker, true
	}
	prefix := len(s) - removed
	out := s[:prefix] + marker
	if len(out) >= len(s) {
		return s, false
	}
	return out, true
}

func findLongestString(v any) (parent any, key string, idx int, ok bool) {
	var bestParent any
	var bestKey string
	var bestIdx int
	bestN := 0
	found := false
	var walk func(any)
	walk = func(node any) {
		switch cur := node.(type) {
		case map[string]any:
			for k, child := range cur {
				if s, isStr := child.(string); isStr && len(s) > bestN {
					bestN = len(s)
					bestParent = cur
					bestKey = k
					bestIdx = 0
					found = true
					continue
				}
				walk(child)
			}
		case []any:
			for i, child := range cur {
				if s, isStr := child.(string); isStr && len(s) > bestN {
					bestN = len(s)
					bestParent = cur
					bestKey = ""
					bestIdx = i
					found = true
					continue
				}
				walk(child)
			}
		}
	}
	walk(v)
	return bestParent, bestKey, bestIdx, found
}

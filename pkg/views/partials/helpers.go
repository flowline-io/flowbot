// Package partials provides HTMX-targeted partial views.
package partials

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// HealthzData is the data model for the health dashboard.
type HealthzData struct {
	PostgresLatency time.Duration
	PostgresOk      bool
	RedisLatency    time.Duration
	RedisOk         bool
	Goroutines      int
	HeapAlloc       uint64
	TotalAlloc      uint64
	SysMem          uint64
	NumGC           uint32
	LastGCPause     time.Duration
	Capabilities    []HealthzCap
	Errors          []flog.ErrorEntry
}

// HealthzCap represents a capability health status for display.
type HealthzCap struct {
	Type   string
	Status string
	Error  string
}

// PageInfo holds pagination state for the event table.
type PageInfo struct {
	Page       int
	TotalPages int
	Total      int64
	PerPage    int
	HasPrev    bool
	HasNext    bool
}

// pageNumbers returns the page numbers to display in pagination.
// Returns a slice where 0 represents an ellipsis.
func pageNumbers(current, total int) []int {
	if total <= 7 {
		nums := make([]int, total)
		for i := range nums {
			nums[i] = i + 1
		}
		return nums
	}
	result := make([]int, 0, 7)
	result = append(result, 1)
	if current-2 > 2 {
		result = append(result, 0)
	}
	start := max(2, current-2)
	end := min(total-1, current+2)
	for i := start; i <= end; i++ {
		result = append(result, i)
	}
	if current+2 < total-1 {
		result = append(result, 0)
	}
	result = append(result, total)
	return result
}

// valuePreview returns a truncated JSON representation of a KV map for display.
func valuePreview(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	s := string(b)
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

// fieldError returns a CSS border color class based on whether the field has a validation error.
func fieldError(errors map[string]string, field string) string {
	if _, ok := errors[field]; ok {
		return "border-red-500"
	}
	return "border-gray-300"
}

// valueJSON returns the full JSON string of a KV map.
func valueJSON(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// configKeyURL returns the key-based URL for a config item.
func configKeyURL(item model.ConfigItem) string {
	return fmt.Sprintf("/service/web/configs/%s/%s/%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configEditURL returns the edit URL for a config item.
func configEditURL(item model.ConfigItem) string {
	return configKeyURL(item) + "/edit"
}

// configRowID returns the DOM element ID for a config row.
func configRowID(item model.ConfigItem) string {
	return fmt.Sprintf("config-%s-%s-%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configFormID returns the DOM element ID for a config form row.
func configFormID(item model.ConfigItem, isNew bool) string {
	if isNew {
		return "config-form-new"
	}
	return "config-form-" + configRowID(item)
}

// cancelURL returns the cancel URL for a config form.
// For new items, returns the list endpoint to refresh the full table and remove the form row.
// For edits, also returns the list endpoint so the row is restored with up-to-date data.
func cancelURL(_ model.ConfigItem, _ bool) string {
	return "/service/web/configs/list"
}

// formatDuration formats a duration for display in the health dashboard.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2f\u00b5s", float64(d.Microseconds())+float64(d.Nanoseconds()%1000)/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds())+float64(d.Microseconds()%1000)/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatBytes formats byte count for display.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

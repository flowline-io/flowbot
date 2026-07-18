// Package wakapi implements the Wakapi coding-stats provider.
package wakapi

// Summary is a Wakapi activity summary.
type Summary struct {
	Projects  []SummaryItem `json:"projects"`
	Languages []SummaryItem `json:"languages"`
	Editors   []SummaryItem `json:"editors"`
	Total     int64         `json:"total_seconds"`
}

// SummaryItem is a named duration bucket in a summary.
type SummaryItem struct {
	Key          string  `json:"key"`
	Total        int64   `json:"total"`
	TotalSeconds float64 `json:"total_seconds"`
	Name         string  `json:"name"`
}

// ProjectsResponse wraps WakaTime-compatible project list.
type ProjectsResponse struct {
	Data []Project `json:"data"`
}

// Project is a Wakapi tracked project.
type Project struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	LastHeartbeatAt string `json:"last_heartbeat_at"`
}

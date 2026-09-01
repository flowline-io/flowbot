// Package wakapi implements the Wakapi coding-stats provider.
package wakapi

const CurrentUser = "current"

// SummaryParams holds query parameters for GET /api/summary.
type SummaryParams struct {
	Interval        string
	From            string
	To              string
	Recompute       *bool
	Project         string
	Language        string
	Editor          string
	OperatingSystem string
	Machine         string
	Label           string
}

// StatsParams holds query parameters for WakaTime-compatible stats.
type StatsParams struct {
	User            string
	Range           string
	Project         string
	Language        string
	Editor          string
	OperatingSystem string
	Machine         string
	Label           string
}

// HeartbeatsParams holds query parameters for listing heartbeats.
type HeartbeatsParams struct {
	User string
	Date string
}

// Summary is the native Wakapi /api/summary response.
type Summary struct {
	UserID           string        `json:"user_id"`
	From             string        `json:"from"`
	To               string        `json:"to"`
	Projects         []SummaryItem `json:"projects"`
	Languages        []SummaryItem `json:"languages"`
	Editors          []SummaryItem `json:"editors"`
	Categories       []SummaryItem `json:"categories"`
	OperatingSystems []SummaryItem `json:"operating_systems"`
	Machines         []SummaryItem `json:"machines"`
	Branches         []SummaryItem `json:"branches"`
	Entities         []SummaryItem `json:"entities"`
	Labels           []SummaryItem `json:"labels"`
}

// SummaryItem is a named duration bucket in a native summary.
type SummaryItem struct {
	Key   string `json:"key"`
	Total int64  `json:"total"`
}

// TotalSeconds sums project totals when the API omits a top-level total.
func (s *Summary) TotalSeconds() int64 {
	if s == nil {
		return 0
	}
	var total int64
	for _, item := range s.Projects {
		total += item.Total
	}
	return total
}

// Stats is WakaTime-compatible activity statistics.
type Stats struct {
	TotalSeconds              float64      `json:"total_seconds"`
	HumanReadableTotal        string       `json:"human_readable_total"`
	HumanReadableRange        string       `json:"human_readable_range"`
	HumanReadableDailyAverage string       `json:"human_readable_daily_average"`
	DailyAverage              float64      `json:"daily_average"`
	Range                     string       `json:"range"`
	Start                     string       `json:"start"`
	End                       string       `json:"end"`
	Status                    string       `json:"status"`
	Timezone                  string       `json:"timezone"`
	UserID                    string       `json:"user_id"`
	Username                  string       `json:"username"`
	Projects                  []StatsEntry `json:"projects"`
	Languages                 []StatsEntry `json:"languages"`
	Editors                   []StatsEntry `json:"editors"`
	Categories                []StatsEntry `json:"categories"`
	OperatingSystems          []StatsEntry `json:"operating_systems"`
	Machines                  []StatsEntry `json:"machines"`
	Branches                  []StatsEntry `json:"branches"`
}

// StatsResponse wraps WakaTime-compatible stats.
type StatsResponse struct {
	Data Stats `json:"data"`
}

// StatsEntry is a named bucket in WakaTime-compatible stats.
type StatsEntry struct {
	Name         string  `json:"name"`
	Seconds      int     `json:"seconds"`
	TotalSeconds float64 `json:"total_seconds"`
	Percent      float64 `json:"percent"`
	Text         string  `json:"text"`
	Hours        int     `json:"hours"`
	Minutes      int     `json:"minutes"`
	Digital      string  `json:"digital"`
}

// AllTime holds cumulative coding time since account creation.
type AllTime struct {
	TotalSeconds float64      `json:"total_seconds"`
	Text         string       `json:"text"`
	IsUpToDate   bool         `json:"is_up_to_date"`
	Range        AllTimeRange `json:"range"`
}

// AllTimeRange is the date span for all-time stats.
type AllTimeRange struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Timezone  string `json:"timezone"`
}

// AllTimeResponse wraps all-time stats.
type AllTimeResponse struct {
	Data AllTime `json:"data"`
}

// ProjectsResponse wraps WakaTime-compatible project list.
type ProjectsResponse struct {
	Data []Project `json:"data"`
}

// Project is a Wakapi tracked project.
type Project struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	LastHeartbeatAt            string `json:"last_heartbeat_at"`
	HumanReadableLastHeartbeat string `json:"human_readable_last_heartbeat_at"`
	URLEncodedName             string `json:"urlencoded_name"`
	CreatedAt                  string `json:"created_at"`
}

// Heartbeat is a single Wakapi heartbeat entry.
type Heartbeat struct {
	ID               string  `json:"id"`
	UserID           string  `json:"user_id"`
	Time             float64 `json:"time"`
	Project          string  `json:"project"`
	Entity           string  `json:"entity"`
	Language         string  `json:"language"`
	Branch           string  `json:"branch"`
	Category         string  `json:"category"`
	IsWrite          bool    `json:"is_write"`
	Type             string  `json:"type"`
	CreatedAt        string  `json:"created_at"`
	LineAdditions    int     `json:"line_additions"`
	LineDeletions    int     `json:"line_deletions"`
	HumanLineChanges int     `json:"human_line_changes"`
}

// HeartbeatsResult wraps heartbeat list for a date.
type HeartbeatsResult struct {
	Data     []Heartbeat `json:"data"`
	Start    string      `json:"start"`
	End      string      `json:"end"`
	Timezone string      `json:"timezone"`
}

// HealthStatus reports Wakapi application and database health.
type HealthStatus struct {
	AppOK bool   `json:"app_ok"`
	DBOK  bool   `json:"db_ok"`
	Raw   string `json:"raw,omitzero"`
}

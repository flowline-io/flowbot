// Package admin defines shared data types for the Admin panel
// (both Wasm frontend and server backend).
// This package contains only pure data types with no business logic
// dependencies, ensuring safe import from both Wasm and server environments.
package admin

import "time"

// ---------------------------------------------------------------------------
// Container management types
// ---------------------------------------------------------------------------

// ContainerStatus represents the runtime state of a container.
type ContainerStatus string

const (
	ContainerRunning ContainerStatus = "running"
	ContainerStopped ContainerStatus = "stopped"
	ContainerPaused  ContainerStatus = "paused"
	ContainerError   ContainerStatus = "error"
)

// Container holds Docker container information.
type Container struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Status    ContainerStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// ContainerCreateRequest is the payload for creating a new container.
type ContainerCreateRequest struct {
	Name   string          `json:"name" validate:"required"`
	Status ContainerStatus `json:"status" validate:"required"`
}

// ContainerUpdateRequest is the payload for updating an existing container.
type ContainerUpdateRequest struct {
	Name   string          `json:"name"`
	Status ContainerStatus `json:"status"`
}

// BatchDeleteRequest is the payload for batch-deleting containers.
type BatchDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required"`
}

// ---------------------------------------------------------------------------
// System settings
// ---------------------------------------------------------------------------

// Settings holds global system configuration.
type Settings struct {
	SiteName       string `json:"site_name"`
	LogoURL        string `json:"logo_url"`
	SEODescription string `json:"seo_description"`
	MaxUploadSize  int64  `json:"max_upload_size"` // bytes
}

// ---------------------------------------------------------------------------
// Pagination & list response
// ---------------------------------------------------------------------------

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// Authentication & user
// ---------------------------------------------------------------------------

// UserInfo represents the currently logged-in user.
type UserInfo struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Platform string `json:"platform"`
}

// SlackOAuthURLResponse wraps the Slack OAuth authorization URL.
type SlackOAuthURLResponse struct {
	URL string `json:"url"`
}

// TokenResponse wraps a token returned after successful login.
type TokenResponse struct {
	Token string `json:"token"`
}

// CodeExchangeRequest is the payload for exchanging a one-time OAuth code
// for a session token.
type CodeExchangeRequest struct {
	Code string `json:"code"`
}

// ---------------------------------------------------------------------------
// Dashboard statistics
// ---------------------------------------------------------------------------

// DashboardStats holds aggregated stats displayed on the admin dashboard.
type DashboardStats struct {
	// Container counts
	TotalContainers   int `json:"total_containers"`
	RunningContainers int `json:"running_containers"`
	StoppedContainers int `json:"stopped_containers"`
	PausedContainers  int `json:"paused_containers"`
	ErrorContainers   int `json:"error_containers"`

	// System info
	Uptime      string `json:"uptime"`       // human-readable uptime
	GoVersion   string `json:"go_version"`   // Go runtime version
	SystemOS    string `json:"system_os"`    // operating system
	SystemArch  string `json:"system_arch"`  // architecture
	NumCPU      int    `json:"num_cpu"`      // number of CPUs
	NumRoutines int    `json:"num_routines"` // number of goroutines
	MemoryUsage uint64 `json:"memory_usage"` // heap memory in bytes
	MemoryTotal uint64 `json:"memory_total"` // total allocated memory in bytes
	Version     string `json:"version"`      // application version

	// Recent containers (last 5)
	RecentContainers []Container `json:"recent_containers"`

	// Activity / event log (recent entries)
	ActivityLog []ActivityEntry `json:"activity_log"`
}

// ActivityEntry represents a single entry in the activity log.
type ActivityEntry struct {
	Time    string `json:"time"`
	Action  string `json:"action"`
	Target  string `json:"target"`
	Success bool   `json:"success"`
}

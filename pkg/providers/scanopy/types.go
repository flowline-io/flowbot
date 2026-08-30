// Package scanopy implements the Scanopy network topology discovery provider.
package scanopy

// Envelope is the standard Scanopy API response wrapper.
type Envelope[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Error   string `json:"error,omitzero"`
	Meta    Meta   `json:"meta"`
}

// Meta holds API and optional pagination metadata.
type Meta struct {
	APIVersion    int             `json:"api_version"`
	ServerVersion string          `json:"server_version"`
	Pagination    *PaginationMeta `json:"pagination,omitzero"`
}

// PaginationMeta describes a paginated list page.
type PaginationMeta struct {
	TotalCount int64 `json:"total_count"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasMore    bool  `json:"has_more"`
}

// VersionInfo is returned by GET /api/version.
type VersionInfo struct {
	APIVersion          int    `json:"api_version"`
	ServerVersion       string `json:"server_version"`
	MinCompatibleClient string `json:"min_compatible_client,omitzero"`
}

// ListParams holds shared list/filter/pagination options.
type ListParams struct {
	NetworkID string
	HostID    string
	Search    string
	// Limit is optional. nil omits the query (API default 50).
	// Use a pointer to 0 for Scanopy "no limit".
	Limit  *int
	Offset int
}

// LimitOf returns a Limit pointer for ListParams.
//
//go:fix inline
func LimitOf(n int) *int {
	return new(n)
}

// Page is a typed list result with pagination metadata.
type Page[T any] struct {
	Items      []T
	Pagination PaginationMeta
	Meta       Meta
}

// Network is a Scanopy network container.
type Network struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	OrganizationID           string   `json:"organization_id,omitzero"`
	StaleAfterHours          *int64   `json:"stale_after_hours,omitzero"`
	EffectiveStaleAfterHours int64    `json:"effective_stale_after_hours,omitzero"`
	Tags                     []string `json:"tags,omitzero"`
	CreatedAt                string   `json:"created_at,omitzero"`
	UpdatedAt                string   `json:"updated_at,omitzero"`
}

// Host is a discovered or manually created network host.
type Host struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	NameSource    string       `json:"name_source,omitzero"`
	Hostname      string       `json:"hostname,omitzero"`
	Description   string       `json:"description,omitzero"`
	NetworkID     string       `json:"network_id"`
	Hidden        bool         `json:"hidden"`
	LastSeenAt    string       `json:"last_seen_at,omitzero"`
	Manufacturer  string       `json:"manufacturer,omitzero"`
	Model         string       `json:"model,omitzero"`
	ManagementURL string       `json:"management_url,omitzero"`
	Source        EntitySource `json:"source,omitzero"`
	Tags          []string     `json:"tags,omitzero"`
	IPAddresses   []IPAddress  `json:"ip_addresses,omitzero"`
	Ports         []Port       `json:"ports,omitzero"`
	Services      []Service    `json:"services,omitzero"`
	CreatedAt     string       `json:"created_at,omitzero"`
	UpdatedAt     string       `json:"updated_at,omitzero"`
}

// EntitySource describes how an entity was created.
type EntitySource struct {
	Type string `json:"type,omitzero"`
}

// IPAddress is an IP assigned to a host.
type IPAddress struct {
	ID         string `json:"id"`
	NetworkID  string `json:"network_id,omitzero"`
	HostID     string `json:"host_id,omitzero"`
	SubnetID   string `json:"subnet_id,omitzero"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address,omitzero"`
	Name       string `json:"name,omitzero"`
	Position   int    `json:"position,omitzero"`
}

// Port is an open port on a host.
type Port struct {
	ID        string `json:"id"`
	HostID    string `json:"host_id,omitzero"`
	NetworkID string `json:"network_id,omitzero"`
	Number    int    `json:"number"`
	Protocol  string `json:"protocol,omitzero"`
	Type      string `json:"type,omitzero"`
}

// Service is a detected or manually added service on a host.
type Service struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	HostID            string       `json:"host_id,omitzero"`
	NetworkID         string       `json:"network_id,omitzero"`
	ServiceDefinition string       `json:"service_definition,omitzero"`
	Position          int          `json:"position,omitzero"`
	Source            EntitySource `json:"source,omitzero"`
	Tags              []string     `json:"tags,omitzero"`
	LastSeenAt        string       `json:"last_seen_at,omitzero"`
	CreatedAt         string       `json:"created_at,omitzero"`
	UpdatedAt         string       `json:"updated_at,omitzero"`
}

// Daemon is a scanning agent.
type Daemon struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	NetworkID     string               `json:"network_id,omitzero"`
	HostID        string               `json:"host_id,omitzero"`
	Mode          string               `json:"mode,omitzero"`
	Version       string               `json:"version,omitzero"`
	LastSeen      string               `json:"last_seen,omitzero"`
	IsUnreachable bool                 `json:"is_unreachable"`
	URL           string               `json:"url,omitzero"`
	VersionStatus *DaemonVersionStatus `json:"version_status,omitzero"`
	CreatedAt     string               `json:"created_at,omitzero"`
	UpdatedAt     string               `json:"updated_at,omitzero"`
}

// DaemonVersionStatus is computed daemon version health.
type DaemonVersionStatus struct {
	Status  string `json:"status,omitzero"`
	Version string `json:"version,omitzero"`
}

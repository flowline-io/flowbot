// Package homelab provides homelab application scanning and registry.
package homelab

// App represents a discovered homelab application with its compose metadata,
// runtime status, and any capabilities derived from labels or probing.
type App struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	ComposeFile string            `json:"compose_file"`
	Services    []ComposeService  `json:"services,omitzero"`
	Networks    []string          `json:"networks,omitzero"`
	Ports       []PortMapping     `json:"ports,omitzero"`
	Labels      map[string]string `json:"labels,omitzero"`
	Status      AppStatus         `json:"status"`
	Health      HealthStatus      `json:"health"`
	// Capabilities discovered from labels and/or probing.
	Capabilities []AppCapability `json:"capabilities,omitzero"`
}

// ComposeService describes a single service entry within a Docker Compose file.
type ComposeService struct {
	Name      string        `json:"name"`
	Image     string        `json:"image,omitzero"`
	Container string        `json:"container,omitzero"`
	Ports     []PortMapping `json:"ports,omitzero"`
}

// PortMapping represents a single host-to-container port binding.
type PortMapping struct {
	Host      string `json:"host,omitzero"`
	HostPort  string `json:"host_port,omitzero"`
	Container string `json:"container,omitzero"`
	Protocol  string `json:"protocol,omitzero"`
}

// AppStatus indicates the aggregate running state of an app's containers.
type AppStatus string

const (
	AppStatusUnknown AppStatus = "unknown"
	AppStatusRunning AppStatus = "running"
	AppStatusStopped AppStatus = "stopped"
	AppStatusPartial AppStatus = "partial"
)

// HealthStatus indicates the observed health of an app.
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// Config controls how the homelab scanner discovers applications and configures
// the runtime and permission model.
type Config struct {
	Root        string
	AppsDir     string
	ComposeFile string
	Allowlist   []string
	Runtime     RuntimeConfig
	Permissions Permissions
	Discovery   DiscoveryConfig
}

// RuntimeConfig selects the runtime mode and its connection parameters.
type RuntimeConfig struct {
	Mode         RuntimeMode
	DockerSocket string
	SSHHost      string
	SSHPort      int
	SSHUser      string
	SSHPassword  string
	SSHKey       string
	SSHHostKey   string
}

// RuntimeMode selects which backend the homelab runtime uses.
type RuntimeMode string

const (
	RuntimeModeNone         RuntimeMode = "none"
	RuntimeModeDockerSocket RuntimeMode = "docker_socket"
	RuntimeModeSSH          RuntimeMode = "ssh"
)

// Permissions defines which container lifecycle operations are allowed.
type Permissions struct {
	Status  bool
	Logs    bool
	Start   bool
	Stop    bool
	Restart bool
	Pull    bool
	Update  bool
	Exec    bool
}

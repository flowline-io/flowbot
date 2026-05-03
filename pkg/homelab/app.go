package homelab

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

type ComposeService struct {
	Name      string        `json:"name"`
	Image     string        `json:"image,omitzero"`
	Container string        `json:"container,omitzero"`
	Ports     []PortMapping `json:"ports,omitzero"`
}

type PortMapping struct {
	Host      string `json:"host,omitzero"`
	HostPort  string `json:"host_port,omitzero"`
	Container string `json:"container,omitzero"`
	Protocol  string `json:"protocol,omitzero"`
}

type AppStatus string

const (
	AppStatusUnknown AppStatus = "unknown"
	AppStatusRunning AppStatus = "running"
	AppStatusStopped AppStatus = "stopped"
	AppStatusPartial AppStatus = "partial"
)

type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type Config struct {
	Root        string
	AppsDir     string
	ComposeFile string
	Allowlist   []string
	Runtime     RuntimeConfig
	Permissions Permissions
	Discovery   DiscoveryConfig
}

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

type RuntimeMode string

const (
	RuntimeModeNone         RuntimeMode = "none"
	RuntimeModeDockerSocket RuntimeMode = "docker_socket"
	RuntimeModeSSH          RuntimeMode = "ssh"
)

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

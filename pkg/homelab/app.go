package homelab

type App struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	ComposeFile string            `json:"compose_file"`
	Services    []ComposeService  `json:"services,omitempty"`
	Networks    []string          `json:"networks,omitempty"`
	Ports       []PortMapping     `json:"ports,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Status      AppStatus         `json:"status"`
	Health      HealthStatus      `json:"health"`
}

type ComposeService struct {
	Name      string        `json:"name"`
	Image     string        `json:"image,omitempty"`
	Container string        `json:"container,omitempty"`
	Ports     []PortMapping `json:"ports,omitempty"`
}

type PortMapping struct {
	Host      string `json:"host,omitempty"`
	HostPort  string `json:"host_port,omitempty"`
	Container string `json:"container,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
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
}

type RuntimeConfig struct {
	Mode         RuntimeMode
	DockerSocket string
	SSHHost      string
	SSHPort      int
	SSHUser      string
	SSHPassword  string
	SSHKey       string
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

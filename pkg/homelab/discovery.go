package homelab

import "time"

// Capability type string constants used for label-based and probe-based
// discovery. These must stay in sync with pkg/hub/capability.go.
const (
	CapBookmark     = "bookmark"
	CapArchive      = "archive"
	CapReader       = "reader"
	CapKanban       = "kanban"
	CapFinance      = "finance"
	CapInfra        = "infra"
	CapShellHistory = "shell_history"
)

// AuthType identifies the authentication mechanism used by an API endpoint.
type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthAPIToken AuthType = "api_token"
	AuthBasic    AuthType = "basic"
	AuthOAuth2   AuthType = "oauth2"
	AuthOIDC     AuthType = "oidc"
	AuthUnknown  AuthType = "unknown"
)

// EndpointInfo describes a discovered API endpoint on an app service.
type EndpointInfo struct {
	BaseURL   string        `json:"base_url"`
	Health    string        `json:"health,omitzero"`
	HealthTTL time.Duration `json:"health_ttl,omitzero"`
	Ports     []int         `json:"ports,omitzero"`
}

// AuthInfo describes the authentication mechanism discovered for an endpoint.
type AuthInfo struct {
	Type        AuthType `json:"type"`
	Header      string   `json:"header,omitzero"`
	Prefix      string   `json:"prefix,omitzero"`
	TokenKey    string   `json:"token_key,omitzero"`
	TokenSource string   `json:"token_source,omitzero"`
}

// AppCapability links a homelab app to a capability and carries discovered
// endpoint and authentication metadata for automatic binding.
type AppCapability struct {
	Capability string        `json:"capability"`
	Backend    string        `json:"backend"`
	Endpoint   *EndpointInfo `json:"endpoint,omitzero"`
	Auth       *AuthInfo     `json:"auth,omitzero"`
}

// DiscoveryConfig controls the behaviour of runtime endpoint discovery probing.
type DiscoveryConfig struct {
	ProbeEnabled       bool          `json:"probe_enabled"`
	ProbeTimeout       time.Duration `json:"probe_timeout"`
	ProbeConcurrency   int           `json:"probe_concurrency"`
	ProbeNetworks      []string      `json:"probe_networks,omitzero"`
	ProbePortStrategy  string        `json:"probe_port_strategy"`
	FingerprintEnabled bool          `json:"fingerprint_enabled"`
	LabelPriority      bool          `json:"label_priority"`
}

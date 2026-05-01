package client

import (
	"context"
)

// HubClient provides access to the hub management API.
type HubClient struct {
	c *Client
}

// HubApp represents a homelab app returned by the hub API.
type HubApp struct {
	Name   string         `json:"name"`
	Path   string         `json:"path"`
	Status string         `json:"status"`
	Health string         `json:"health"`
	Labels map[string]any `json:"labels,omitempty"`
}

// HubAppStatus represents the status of a homelab app.
type HubAppStatus struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Logs   []string `json:"logs,omitempty"`
}

// HubCapability represents a registered capability.
type HubCapability struct {
	Type        string `json:"type"`
	Backend     string `json:"backend"`
	App         string `json:"app"`
	Description string `json:"description,omitempty"`
	Healthy     bool   `json:"healthy"`
}

// HubHealth represents the overall hub health.
type HubHealth struct {
	Status      string             `json:"status"`
	Timestamp   string             `json:"timestamp"`
	Details     []CapabilityHealth `json:"details,omitempty"`
	AppStatuses []AppHealthStatus  `json:"app_statuses,omitempty"`
}

// CapabilityHealth represents a single capability's health.
type CapabilityHealth struct {
	Capability  string `json:"capability"`
	Backend     string `json:"backend"`
	App         string `json:"app"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// AppHealthStatus represents a homelab app's health.
type AppHealthStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Health string `json:"health"`
}

// ListApps lists all registered homelab apps.
func (h *HubClient) ListApps(ctx context.Context) ([]HubApp, error) {
	var result []HubApp
	err := h.c.Get(ctx, "/hub/apps", &result)
	return result, err
}

// GetApp returns a single app by name.
func (h *HubClient) GetApp(ctx context.Context, name string) (*HubApp, error) {
	var result HubApp
	err := h.c.Get(ctx, "/hub/apps/"+name, &result)
	return &result, err
}

// GetAppStatus returns the status of an app.
func (h *HubClient) GetAppStatus(ctx context.Context, name string) (*HubAppStatus, error) {
	var result HubAppStatus
	err := h.c.Get(ctx, "/hub/apps/"+name+"/status", &result)
	return &result, err
}

// GetAppLogs returns logs for an app.
func (h *HubClient) GetAppLogs(ctx context.Context, name string, tail int) (*HubAppStatus, error) {
	path := "/hub/apps/" + name + "/logs"
	if tail > 0 {
		path += "?tail=" + intToString(tail)
	}
	var result HubAppStatus
	err := h.c.Get(ctx, path, &result)
	return &result, err
}

// ListCapabilities lists all registered capabilities.
func (h *HubClient) ListCapabilities(ctx context.Context) ([]HubCapability, error) {
	var result []HubCapability
	err := h.c.Get(ctx, "/hub/capabilities", &result)
	return result, err
}

// GetCapability returns a single capability by type.
func (h *HubClient) GetCapability(ctx context.Context, capType string) (*HubCapability, error) {
	var result HubCapability
	err := h.c.Get(ctx, "/hub/capabilities/"+capType, &result)
	return &result, err
}

// GetHealth returns the overall hub health.
func (h *HubClient) GetHealth(ctx context.Context) (*HubHealth, error) {
	var result HubHealth
	err := h.c.Get(ctx, "/hub/health", &result)
	return &result, err
}

// StartApp starts a homelab app.
func (h *HubClient) StartApp(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	err := h.c.Post(ctx, "/hub/apps/"+name+"/start", nil, &result)
	return result, err
}

// StopApp stops a homelab app.
func (h *HubClient) StopApp(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	err := h.c.Post(ctx, "/hub/apps/"+name+"/stop", nil, &result)
	return result, err
}

// RestartApp restarts a homelab app.
func (h *HubClient) RestartApp(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	err := h.c.Post(ctx, "/hub/apps/"+name+"/restart", nil, &result)
	return result, err
}

// intToString converts an int to string.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

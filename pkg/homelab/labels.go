package homelab

import (
	"strings"
	"time"
)

// Label key constants for the flowbot docker-compose label convention.
const (
	LabelCapability        = "flowbot.capability"
	LabelBackend           = "flowbot.backend"
	LabelEndpointBase      = "flowbot.endpoint.base"
	LabelEndpointHealth    = "flowbot.endpoint.health"
	LabelEndpointHealthTTL = "flowbot.endpoint.health_ttl"
	LabelAuthType          = "flowbot.auth.type"
	LabelAuthHeader        = "flowbot.auth.header"
	LabelAuthPrefix        = "flowbot.auth.prefix"
	LabelAuthTokenKey      = "flowbot.auth.token_key"
	LabelAuthTokenSource   = "flowbot.auth.token_source"
)

// knownCapabilities maps label values to capability type constants.
var knownCapabilities = map[string]string{
	"bookmark":      CapBookmark,
	"archive":       CapArchive,
	"reader":        CapReader,
	"kanban":        CapKanban,
	"finance":       CapFinance,
	"infra":         CapInfra,
	"shell_history": CapShellHistory,
}

// ParseLabels extracts AppCapability entries from the labels map using the
// flowbot label convention. Returns nil when no capability label is present
// or the capability value is not recognised.
func ParseLabels(labels map[string]string) []AppCapability {
	if len(labels) == 0 {
		return nil
	}

	capLabel := strings.TrimSpace(labels[LabelCapability])
	backendLabel := strings.TrimSpace(labels[LabelBackend])

	capType, ok := knownCapabilities[capLabel]
	if !ok {
		return nil
	}
	if backendLabel == "" {
		backendLabel = capLabel
	}

	capability := AppCapability{
		Capability: capType,
		Backend:    backendLabel,
	}

	if baseURL := strings.TrimSpace(labels[LabelEndpointBase]); baseURL != "" {
		endpoint := &EndpointInfo{
			BaseURL: baseURL,
			Health:  strings.TrimSpace(labels[LabelEndpointHealth]),
		}
		if ttl := strings.TrimSpace(labels[LabelEndpointHealthTTL]); ttl != "" {
			if d, err := time.ParseDuration(ttl); err == nil {
				endpoint.HealthTTL = d
			}
		}
		capability.Endpoint = endpoint
	}

	authType := AuthType(strings.TrimSpace(labels[LabelAuthType]))
	if authType != "" && authType != AuthNone {
		auth := &AuthInfo{
			Type:        authType,
			Header:      strings.TrimSpace(labels[LabelAuthHeader]),
			Prefix:      strings.TrimSpace(labels[LabelAuthPrefix]),
			TokenKey:    strings.TrimSpace(labels[LabelAuthTokenKey]),
			TokenSource: strings.TrimSpace(labels[LabelAuthTokenSource]),
		}
		capability.Auth = auth
	}

	return []AppCapability{capability}
}

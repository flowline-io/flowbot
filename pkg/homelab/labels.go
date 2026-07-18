package homelab

import (
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// Label key constants for the flowbot docker-compose label convention.
const (
	LabelCapability        = "flowbot.capability"
	LabelBackend           = "flowbot.backend" // deprecated: ignored; CapType is provider ID
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
// Legacy domain labels are mapped for this release only.
var knownCapabilities = map[string]string{
	// Provider IDs (canonical)
	"karakeep": CapKarakeep,
	"miniflux": CapMiniflux,
	"kanboard": CapKanboard,
	"trilium":  CapTrilium,
	"memos":    CapMemos,
	"gitea":    CapGitea,
	"github":   CapGithub,
	"example":  CapExample,
	"devops":   CapDevops,
	// Discovery-only
	"archive":       CapArchive,
	"finance":       CapFinance,
	"infra":         CapInfra,
	"shell_history": CapShellHistory,
	// Legacy domain labels (deprecated this version)
	"bookmark": CapKarakeep,
	"reader":   CapMiniflux,
	"kanban":   CapKanboard,
	"note":     CapTrilium,
	"memo":     CapMemos,
	"forge":    CapGitea,
}

var legacyCapabilityLabels = map[string]string{
	"bookmark": CapKarakeep,
	"reader":   CapMiniflux,
	"kanban":   CapKanboard,
	"note":     CapTrilium,
	"memo":     CapMemos,
	"forge":    CapGitea,
}

// ParseLabels extracts AppCapability entries from the labels map using the
// flowbot label convention. Returns nil when no capability label is present
// or the capability value is not recognised.
func ParseLabels(labels map[string]string) []AppCapability {
	if len(labels) == 0 {
		return nil
	}

	capLabel := strings.TrimSpace(labels[LabelCapability])
	capType, ok := knownCapabilities[capLabel]
	if !ok {
		return nil
	}
	if mapped, legacy := legacyCapabilityLabels[capLabel]; legacy {
		flog.Warn("homelab: deprecated flowbot.capability=%q; use %q (legacy mapping removed next release)", capLabel, mapped)
		capType = mapped
	}
	if strings.TrimSpace(labels[LabelBackend]) != "" {
		flog.Warn("homelab: flowbot.backend label is deprecated and ignored")
	}

	capability := AppCapability{
		Capability: capType,
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

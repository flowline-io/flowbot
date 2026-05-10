package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels_NilOrEmpty(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{name: "nil labels", labels: nil},
		{name: "empty labels", labels: map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, ParseLabels(tt.labels))
		})
	}
}

func TestParseLabels_UnknownCapability(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "unknown capability returns nil",
			labels: map[string]string{
				LabelCapability: "nonexistent",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, ParseLabels(tt.labels))
		})
	}
}

func TestParseLabels_MinimalLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "minimal capability label only",
			labels: map[string]string{
				LabelCapability: "bookmark",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, CapBookmark, capabilities[0].Capability)
			assert.Equal(t, "bookmark", capabilities[0].Backend)
			assert.Nil(t, capabilities[0].Endpoint)
			assert.Nil(t, capabilities[0].Auth)
		})
	}
}

func TestParseLabels_WithBackend(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "capability with custom backend",
			labels: map[string]string{
				LabelCapability: "bookmark",
				LabelBackend:    "karakeep",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, CapBookmark, capabilities[0].Capability)
			assert.Equal(t, "karakeep", capabilities[0].Backend)
		})
	}
}

func TestParseLabels_FullEndpointInfo(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "full endpoint metadata",
			labels: map[string]string{
				LabelCapability:        "archive",
				LabelEndpointBase:      "http://archivebox:8000",
				LabelEndpointHealth:    "/health",
				LabelEndpointHealthTTL: "30s",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, CapArchive, capabilities[0].Capability)
			require.NotNil(t, capabilities[0].Endpoint)
			assert.Equal(t, "http://archivebox:8000", capabilities[0].Endpoint.BaseURL)
			assert.Equal(t, "/health", capabilities[0].Endpoint.Health)
			assert.InDelta(t, float64(30000000000), float64(capabilities[0].Endpoint.HealthTTL), float64(1000000))
			assert.Nil(t, capabilities[0].Auth)
		})
	}
}

func TestParseLabels_FullAuthInfo(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "full auth metadata",
			labels: map[string]string{
				LabelCapability:      "kanban",
				LabelAuthType:        "api_token",
				LabelAuthHeader:      "X-API-Key",
				LabelAuthPrefix:      "",
				LabelAuthTokenKey:    "KANBOARD_API_KEY",
				LabelAuthTokenSource: "env",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, CapKanban, capabilities[0].Capability)
			require.NotNil(t, capabilities[0].Auth)
			assert.Equal(t, AuthAPIToken, capabilities[0].Auth.Type)
			assert.Equal(t, "X-API-Key", capabilities[0].Auth.Header)
			assert.Empty(t, capabilities[0].Auth.Prefix)
			assert.Equal(t, "KANBOARD_API_KEY", capabilities[0].Auth.TokenKey)
			assert.Equal(t, "env", capabilities[0].Auth.TokenSource)
		})
	}
}

func TestParseLabels_AuthNoneSkipped(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "auth type none is not included",
			labels: map[string]string{
				LabelCapability: "finance",
				LabelAuthType:   "none",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Nil(t, capabilities[0].Auth)
		})
	}
}

func TestParseLabels_AllCapabilities(t *testing.T) {
	tests := []struct {
		labelValue string
		expected   string
	}{
		{"bookmark", CapBookmark},
		{"archive", CapArchive},
		{"reader", CapReader},
		{"kanban", CapKanban},
		{"finance", CapFinance},
		{"infra", CapInfra},
		{"shell_history", CapShellHistory},
	}
	for _, tt := range tests {
		t.Run(tt.labelValue, func(t *testing.T) {
			labels := map[string]string{LabelCapability: tt.labelValue}
			capabilities := ParseLabels(labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tt.expected, capabilities[0].Capability)
		})
	}
}

func TestParseLabels_TrimSpaces(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "trims surrounding whitespace",
			labels: map[string]string{
				LabelCapability: "  bookmark  ",
				LabelBackend:    "  karakeep  ",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, "karakeep", capabilities[0].Backend)
		})
	}
}

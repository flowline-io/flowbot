package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels_NilOrEmpty(t *testing.T) {
	assert.Nil(t, ParseLabels(nil))
	assert.Nil(t, ParseLabels(map[string]string{}))
}

func TestParseLabels_UnknownCapability(t *testing.T) {
	labels := map[string]string{
		LabelCapability: "nonexistent",
	}
	assert.Nil(t, ParseLabels(labels))
}

func TestParseLabels_MinimalLabel(t *testing.T) {
	labels := map[string]string{
		LabelCapability: "bookmark",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Equal(t, CapBookmark, capabilities[0].Capability)
	assert.Equal(t, "bookmark", capabilities[0].Backend)
	assert.Nil(t, capabilities[0].Endpoint)
	assert.Nil(t, capabilities[0].Auth)
}

func TestParseLabels_WithBackend(t *testing.T) {
	labels := map[string]string{
		LabelCapability: "bookmark",
		LabelBackend:    "karakeep",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Equal(t, CapBookmark, capabilities[0].Capability)
	assert.Equal(t, "karakeep", capabilities[0].Backend)
}

func TestParseLabels_FullEndpointInfo(t *testing.T) {
	labels := map[string]string{
		LabelCapability:        "archive",
		LabelEndpointBase:      "http://archivebox:8000",
		LabelEndpointHealth:    "/health",
		LabelEndpointHealthTTL: "30s",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Equal(t, CapArchive, capabilities[0].Capability)
	require.NotNil(t, capabilities[0].Endpoint)
	assert.Equal(t, "http://archivebox:8000", capabilities[0].Endpoint.BaseURL)
	assert.Equal(t, "/health", capabilities[0].Endpoint.Health)
	assert.InDelta(t, float64(30000000000), float64(capabilities[0].Endpoint.HealthTTL), float64(1000000))
	assert.Nil(t, capabilities[0].Auth)
}

func TestParseLabels_FullAuthInfo(t *testing.T) {
	labels := map[string]string{
		LabelCapability:      "kanban",
		LabelAuthType:        "api_token",
		LabelAuthHeader:      "X-API-Key",
		LabelAuthPrefix:      "",
		LabelAuthTokenKey:    "KANBOARD_API_KEY",
		LabelAuthTokenSource: "env",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Equal(t, CapKanban, capabilities[0].Capability)
	require.NotNil(t, capabilities[0].Auth)
	assert.Equal(t, AuthAPIToken, capabilities[0].Auth.Type)
	assert.Equal(t, "X-API-Key", capabilities[0].Auth.Header)
	assert.Equal(t, "", capabilities[0].Auth.Prefix)
	assert.Equal(t, "KANBOARD_API_KEY", capabilities[0].Auth.TokenKey)
	assert.Equal(t, "env", capabilities[0].Auth.TokenSource)
}

func TestParseLabels_AuthNoneSkipped(t *testing.T) {
	labels := map[string]string{
		LabelCapability: "finance",
		LabelAuthType:   "none",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Nil(t, capabilities[0].Auth)
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
	for _, tc := range tests {
		t.Run(tc.labelValue, func(t *testing.T) {
			labels := map[string]string{LabelCapability: tc.labelValue}
			capabilities := ParseLabels(labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tc.expected, capabilities[0].Capability)
		})
	}
}

func TestParseLabels_TrimSpaces(t *testing.T) {
	labels := map[string]string{
		LabelCapability: "  bookmark  ",
		LabelBackend:    "  karakeep  ",
	}
	capabilities := ParseLabels(labels)
	require.Len(t, capabilities, 1)
	assert.Equal(t, "karakeep", capabilities[0].Backend)
}

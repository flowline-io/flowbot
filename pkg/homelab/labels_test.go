package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels_NilOrEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{name: "nil labels", labels: nil},
		{name: "empty labels", labels: map[string]string{}},
		{name: "only non-capability keys", labels: map[string]string{"env": "prod", "tier": "frontend"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, ParseLabels(tt.labels))
		})
	}
}

func TestParseLabels_UnknownCapability(t *testing.T) {
	t.Parallel()
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
		{
			name: "random capability string",
			labels: map[string]string{
				LabelCapability: "monitoring",
			},
		},
		{
			name: "unknown capability with valid endpoint ignored",
			labels: map[string]string{
				LabelCapability:     "something_else",
				LabelEndpointBase:   "http://localhost:9999",
				LabelEndpointHealth: "/ping",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, ParseLabels(tt.labels))
		})
	}
}

func TestParseLabels_MinimalLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "minimal bookmark capability",
			labels: map[string]string{
				LabelCapability: "bookmark",
			},
		},
		{
			name: "minimal archive capability",
			labels: map[string]string{
				LabelCapability: "archive",
			},
		},
		{
			name: "minimal finance capability",
			labels: map[string]string{
				LabelCapability: "finance",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tt.labels[LabelCapability], capabilities[0].Backend)
			assert.Nil(t, capabilities[0].Endpoint)
			assert.Nil(t, capabilities[0].Auth)
		})
	}
}

func TestParseLabels_WithBackend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "bookmark with karakeep backend",
			labels:   map[string]string{LabelCapability: "bookmark", LabelBackend: "karakeep"},
			expected: "karakeep",
		},
		{
			name:     "infra with portainer backend",
			labels:   map[string]string{LabelCapability: "infra", LabelBackend: "portainer"},
			expected: "portainer",
		},
		{
			name:     "finance with actual-budget backend",
			labels:   map[string]string{LabelCapability: "finance", LabelBackend: "actual-budget"},
			expected: "actual-budget",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tt.expected, capabilities[0].Backend)
		})
	}
}

func TestParseLabels_FullEndpointInfo(t *testing.T) {
	t.Parallel()
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
		{
			name: "endpoint with health path but no ttl",
			labels: map[string]string{
				LabelCapability:     "reader",
				LabelEndpointBase:   "https://reader.example.com:443",
				LabelEndpointHealth: "/api/health",
			},
		},
		{
			name: "endpoint with invalid ttl ignored",
			labels: map[string]string{
				LabelCapability:        "kanban",
				LabelEndpointBase:      "http://kanboard:80",
				LabelEndpointHealthTTL: "not-a-duration",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			require.NotNil(t, capabilities[0].Endpoint)
			assert.Equal(t, tt.labels[LabelEndpointBase], capabilities[0].Endpoint.BaseURL)
			assert.Equal(t, tt.labels[LabelEndpointHealth], capabilities[0].Endpoint.Health)
		})
	}
}

func TestParseLabels_FullAuthInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "full auth metadata for api_token",
			labels: map[string]string{
				LabelCapability:      "kanban",
				LabelAuthType:        "api_token",
				LabelAuthHeader:      "X-API-Key",
				LabelAuthPrefix:      "",
				LabelAuthTokenKey:    "KANBOARD_API_KEY",
				LabelAuthTokenSource: "env",
			},
		},
		{
			name: "auth type basic with prefix",
			labels: map[string]string{
				LabelCapability:      "infra",
				LabelAuthType:        "basic",
				LabelAuthHeader:      "Authorization",
				LabelAuthPrefix:      "Basic",
				LabelAuthTokenKey:    "INFRA_CREDS",
				LabelAuthTokenSource: "secret",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			require.NotNil(t, capabilities[0].Auth)
			assert.Equal(t, AuthType(tt.labels[LabelAuthType]), capabilities[0].Auth.Type)
			assert.Equal(t, tt.labels[LabelAuthHeader], capabilities[0].Auth.Header)
			assert.Equal(t, tt.labels[LabelAuthPrefix], capabilities[0].Auth.Prefix)
			assert.Equal(t, tt.labels[LabelAuthTokenKey], capabilities[0].Auth.TokenKey)
			assert.Equal(t, tt.labels[LabelAuthTokenSource], capabilities[0].Auth.TokenSource)
		})
	}
}

func TestParseLabels_AuthNoneSkipped(t *testing.T) {
	t.Parallel()
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
		{
			name: "auth type none with other auth fields populated",
			labels: map[string]string{
				LabelCapability:      "kanban",
				LabelAuthType:        "none",
				LabelAuthHeader:      "Authorization",
				LabelAuthTokenKey:    "IGNORED_KEY",
				LabelAuthTokenSource: "env",
			},
		},
		{
			name: "no auth type label at all",
			labels: map[string]string{
				LabelCapability: "archive",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Nil(t, capabilities[0].Auth)
		})
	}
}

func TestParseLabels_AllCapabilities(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			labels := map[string]string{LabelCapability: tt.labelValue}
			capabilities := ParseLabels(labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tt.expected, capabilities[0].Capability)
		})
	}
}

func TestParseLabels_TrimSpaces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		labels          map[string]string
		expectedBackend string
	}{
		{
			name: "trims surrounding whitespace",
			labels: map[string]string{
				LabelCapability: "  bookmark  ",
				LabelBackend:    "  karakeep  ",
			},
			expectedBackend: "karakeep",
		},
		{
			name: "trims tabs and newlines",
			labels: map[string]string{
				LabelCapability: "\treader\n",
				LabelBackend:    "\tminiflux\n",
			},
			expectedBackend: "miniflux",
		},
		{
			name: "whitespace-only backend falls back to capability name",
			labels: map[string]string{
				LabelCapability: "infra",
				LabelBackend:    "   ",
			},
			expectedBackend: "infra",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capabilities := ParseLabels(tt.labels)
			require.Len(t, capabilities, 1)
			assert.Equal(t, tt.expectedBackend, capabilities[0].Backend)
		})
	}
}

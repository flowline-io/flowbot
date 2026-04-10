package adguard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdGuardHome_GetStatus(t *testing.T) {
	tests := []struct {
		name       string
		response   ServerStatus
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful status retrieval",
			response: ServerStatus{
				DnsAddresses:      []string{"127.0.0.1", "::1"},
				DnsPort:           53,
				HttpPort:          80,
				ProtectionEnabled: true,
				Running:           true,
				Version:           "v0.107.0",
				Language:          "en",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/status", r.URL.Path)
				assert.Equal(t, "Basic YWRtaW46cGFzc3dvcmQ=", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				err := json.NewEncoder(w).Encode(tt.response)
				require.NoError(t, err)
			}))
			defer server.Close()

			client := NewAdGuardHome(server.URL, "admin", "password")
			result, err := client.GetStatus()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.Version, result.Version)
				assert.Equal(t, tt.response.ProtectionEnabled, result.ProtectionEnabled)
				assert.Equal(t, tt.response.Running, result.Running)
			}
		})
	}
}

func TestAdGuardHome_GetStats(t *testing.T) {
	timeUnits := "hours"
	numQueries := int32(1000)
	numBlocked := int32(100)
	avgTime := float32(0.5)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/stats", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		response := Stats{
			TimeUnits:             &timeUnits,
			NumDnsQueries:         &numQueries,
			NumBlockedFiltering:   &numBlocked,
			AvgProcessingTime:     &avgTime,
			TopQueriedDomains:     []TopArrayEntry{},
			TopClients:            []TopArrayEntry{},
			TopBlockedDomains:     []TopArrayEntry{},
			TopUpstreamsResponses: []TopArrayEntry{},
			TopUpstreamsAvgTime:   []TopArrayEntry{},
			DnsQueries:            []int32{10, 20, 30},
			BlockedFiltering:      []int32{1, 2, 3},
			ReplacedSafebrowsing:  []int32{},
			ReplacedParental:      []int32{},
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewAdGuardHome(server.URL, "admin", "password")
	result, err := client.GetStats()

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "hours", *result.TimeUnits)
	assert.Equal(t, int32(1000), *result.NumDnsQueries)
	assert.Equal(t, int32(100), *result.NumBlockedFiltering)
	assert.Equal(t, float32(0.5), *result.AvgProcessingTime)
	assert.Len(t, result.DnsQueries, 3)
}

func TestNewAdGuardHome(t *testing.T) {
	client := NewAdGuardHome("http://localhost:3000", "admin", "pass")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}

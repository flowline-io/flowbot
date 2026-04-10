package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudflare_GetAnalytics(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		start      string
		end        string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful analytics query",
			start:      "2024-01-01T00:00:00Z",
			end:        "2024-01-02T00:00:00Z",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "api error",
			start:      "2024-01-01T00:00:00Z",
			end:        "2024-01-02T00:00:00Z",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/graphql", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, _ := io.ReadAll(r.Body)
				var reqBody map[string]any
				json.Unmarshal(body, &reqBody)
				assert.NotNil(t, reqBody["query"])

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					response := map[string]any{
						"data": map[string]any{
							"viewer": map[string]any{
								"zones": []map[string]any{
									{
										"firewallEventsAdaptive": []map[string]any{
											{
												"action":                "block",
												"clientRequestHTTPHost": "example.com",
												"datetime":              now.Format(time.RFC3339),
												"rayName":               "7abc123",
												"userAgent":             "Mozilla/5.0",
											},
										},
									},
								},
							},
						},
						"errors": nil,
					}
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			client := NewCloudflare("test-token", "zone-123")
			client.c.SetBaseURL(server.URL + "/")

			result, err := client.GetAnalytics(tt.start, tt.end)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCloudflare_GetAnalytics_ZoneIDInjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var reqBody map[string]any
		err = json.Unmarshal(body, &reqBody)
		require.NoError(t, err)

		query, ok := reqBody["query"].(string)
		require.True(t, ok)
		assert.Contains(t, query, "test-zone-id")
		assert.Contains(t, query, "2024-01-01T00:00:00Z")
		assert.Contains(t, query, "2024-01-02T00:00:00Z")

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"viewer": map[string]any{"zones": []map[string]any{}}}})
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewCloudflare("test-token", "test-zone-id")
	client.c.SetBaseURL(server.URL + "/")

	_, err := client.GetAnalytics("2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z")
	require.NoError(t, err)
}

func TestCloudflare_AuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewCloudflare("invalid-token", "zone-123")
	client.c.SetBaseURL(server.URL + "/")

	_, err := client.GetAnalytics("2024-01-01", "2024-01-02")
	assert.Error(t, err)
}

func TestNewCloudflare(t *testing.T) {
	client := NewCloudflare("my-token", "my-zone")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
	assert.Equal(t, "my-token", client.token)
	assert.Equal(t, "my-zone", client.zoneID)
}

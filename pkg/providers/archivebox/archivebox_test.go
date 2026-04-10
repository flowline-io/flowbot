package archivebox

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveBox_Add(t *testing.T) {
	tests := []struct {
		name       string
		request    Data
		response   Response
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful archive addition",
			request: Data{
				Urls:       []string{"https://example.com"},
				Tag:        "test",
				Depth:      0,
				Update:     false,
				UpdateAll:  false,
				IndexOnly:  false,
				Overwrite:  false,
				Init:       false,
				Extractors: "",
				Parser:     "",
			},
			response: Response{
				Success: true,
				Errors:  []string{},
				Result:  []string{"https://example.com"},
				Stdout:  "Added 1 URL",
				Stderr:  "",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "archive addition with errors",
			request: Data{
				Urls: []string{"invalid-url"},
			},
			response: Response{
				Success: false,
				Errors:  []string{"Invalid URL format"},
				Result:  []string{},
				Stdout:  "",
				Stderr:  "Error processing URL",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/cli/add", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var reqData Data
				err = json.Unmarshal(body, &reqData)
				require.NoError(t, err)
				assert.Equal(t, tt.request.Urls, reqData.Urls)

				w.WriteHeader(tt.statusCode)
				err = json.NewEncoder(w).Encode(tt.response)
				require.NoError(t, err)
			}))
			defer server.Close()

			client := NewArchiveBox(server.URL, "test-token")
			result, err := client.Add(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.Success, result.Success)
				assert.Equal(t, tt.response.Stdout, result.Stdout)
				assert.Equal(t, tt.response.Stderr, result.Stderr)
			}
		})
	}
}

func TestArchiveBox_AddMultipleUrls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var reqData Data
		err = json.Unmarshal(body, &reqData)
		require.NoError(t, err)

		assert.Len(t, reqData.Urls, 3)

		response := Response{
			Success: true,
			Result:  reqData.Urls,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewArchiveBox(server.URL, "test-token")
	data := Data{
		Urls: []string{
			"https://example1.com",
			"https://example2.com",
			"https://example3.com",
		},
	}
	result, err := client.Add(data)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestNewArchiveBox(t *testing.T) {
	client := NewArchiveBox("http://localhost:8000", "my-token")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}

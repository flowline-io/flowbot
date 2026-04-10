package slash

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlash_CreateShortcut(t *testing.T) {
	tests := []struct {
		name       string
		shortcut   Shortcut
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful creation",
			shortcut: Shortcut{
				Name:        "test-shortcut",
				Link:        "https://example.com",
				Title:       "Test Title",
				Description: "Test Description",
				Tags:        []string{"test", "example"},
				Visibility:  "PUBLIC",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "bad request",
			shortcut: Shortcut{
				Name: "",
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "unauthorized",
			shortcut: Shortcut{
				Name: "test",
			},
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/shortcuts", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")

				body, _ := io.ReadAll(r.Body)
				var reqData Shortcut
				err := json.Unmarshal(body, &reqData)
				require.NoError(t, err)
				assert.Equal(t, tt.shortcut.Name, reqData.Name)
				assert.Equal(t, tt.shortcut.Link, reqData.Link)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewSlash(server.URL, "test-token")
			err := client.CreateShortcut(tt.shortcut)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSlash_UpdateShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/shortcuts/123"
		assert.Equal(t, expectedPath, r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var reqData Shortcut
		err = json.Unmarshal(body, &reqData)
		require.NoError(t, err)

		assert.Equal(t, int32(123), reqData.Id)
		assert.Equal(t, "updated-shortcut", reqData.Name)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSlash(server.URL, "test-token")
	shortcut := Shortcut{
		Id:   123,
		Name: "updated-shortcut",
		Link: "https://updated.example.com",
	}
	err := client.UpdateShortcut(shortcut)

	require.NoError(t, err)
}

func TestSlash_DeleteShortcut(t *testing.T) {
	tests := []struct {
		name       string
		id         int32
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			id:         123,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "not found",
			id:         999,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/shortcuts/%d", tt.id)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewSlash(server.URL, "test-token")
			err := client.DeleteShortcut(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSlash_GetShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/shortcuts/456", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")

		response := Shortcut{
			Id:          456,
			CreatorId:   1,
			CreatedTime: time.Now(),
			UpdatedTime: time.Now(),
			Name:        "my-shortcut",
			Link:        "https://example.com/resource",
			Title:       "Example Resource",
			Tags:        []string{"docs", "reference"},
			Description: "A useful resource",
			Visibility:  "PUBLIC",
			ViewCount:   100,
			OGMetaData: struct {
				Title       string `json:"title,omitempty"`
				Description string `json:"description,omitempty"`
				Image       string `json:"image,omitempty"`
			}{
				Title:       "Example Resource",
				Description: "Resource description",
				Image:       "https://example.com/image.png",
			},
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewSlash(server.URL, "test-token")
	result, err := client.GetShortcut(456)

	require.NoError(t, err)
	assert.Equal(t, int32(456), result.Id)
	assert.Equal(t, "my-shortcut", result.Name)
	assert.Equal(t, "https://example.com/resource", result.Link)
	assert.Equal(t, int32(100), result.ViewCount)
	assert.Equal(t, "Example Resource", result.OGMetaData.Title)
}

func TestSlash_GetShortcut_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewSlash(server.URL, "test-token")
	_, err := client.GetShortcut(999)

	assert.Error(t, err)
}

func TestSlash_ListShortcuts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/shortcuts", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")

		// Return array of pointers as expected by the API
		shortcuts := []*Shortcut{
			{
				Id:          1,
				Name:        "shortcut-1",
				Link:        "https://example1.com",
				Title:       "Example 1",
				Description: "First shortcut",
				ViewCount:   50,
			},
			{
				Id:          2,
				Name:        "shortcut-2",
				Link:        "https://example2.com",
				Title:       "Example 2",
				Description: "Second shortcut",
				ViewCount:   100,
			},
			{
				Id:          3,
				Name:        "shortcut-3",
				Link:        "https://example3.com",
				Title:       "Example 3",
				Description: "Third shortcut",
				ViewCount:   150,
			},
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(shortcuts)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewSlash(server.URL, "test-token")
	result, err := client.ListShortcuts()

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "shortcut-1", result[0].Name)
	assert.Equal(t, "shortcut-2", result[1].Name)
	assert.Equal(t, "shortcut-3", result[2].Name)
}

func TestSlash_ListShortcuts_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shortcuts := []*Shortcut{}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(shortcuts)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewSlash(server.URL, "test-token")
	result, err := client.ListShortcuts()

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNewSlash(t *testing.T) {
	client := NewSlash("https://slash.example.com", "my-token")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
	assert.Equal(t, "my-token", client.token)
}

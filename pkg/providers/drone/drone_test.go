package drone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrone_CreateBuild(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		repoName   string
		response   Build
		statusCode int
		wantErr    bool
	}{
		{
			name:      "successful build creation",
			namespace: "myorg",
			repoName:  "myrepo",
			response: Build{
				ID:           123,
				RepoID:       456,
				Number:       1,
				Status:       "pending",
				Event:        "push",
				Action:       "",
				Link:         "https://drone.example.com/myorg/myrepo/1",
				Message:      "Test commit",
				Before:       "abc123",
				After:        "def456",
				Ref:          "refs/heads/main",
				SourceRepo:   "myorg/myrepo",
				Source:       "main",
				Target:       "main",
				AuthorLogin:  "testuser",
				AuthorName:   "Test User",
				AuthorEmail:  "test@example.com",
				AuthorAvatar: "https://example.com/avatar.png",
				Sender:       "testuser",
				Started:      0,
				Finished:     0,
				Created:      1234567890,
				Updated:      1234567890,
				Version:      1,
				Stages:       []Stage{},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/repos/" + tt.namespace + "/" + tt.repoName + "/builds"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewDrone(server.URL, "test-token")
			result, err := client.CreateBuild(tt.namespace, tt.repoName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.ID, result.ID)
				assert.Equal(t, tt.response.Number, result.Number)
				assert.Equal(t, tt.response.Status, result.Status)
				assert.Equal(t, tt.response.AuthorLogin, result.AuthorLogin)
			}
		})
	}
}

func TestDrone_CreateBuildWithStages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := Build{
			ID:     123,
			Number: 1,
			Status: "running",
			Stages: []Stage{
				{
					ID:        1,
					RepoID:    456,
					BuildID:   123,
					Number:    1,
					Name:      "build",
					Kind:      "pipeline",
					Type:      "docker",
					Status:    "running",
					ErrIgnore: false,
					ExitCode:  0,
					Machine:   "drone-runner",
					OS:        "linux",
					Arch:      "amd64",
					Started:   1234567890,
					Stopped:   0,
					Created:   1234567890,
					Updated:   1234567890,
					Version:   1,
					OnSuccess: true,
					OnFailure: false,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewDrone(server.URL, "test-token")
	result, err := client.CreateBuild("myorg", "myrepo")

	require.NoError(t, err)
	assert.Len(t, result.Stages, 1)
	assert.Equal(t, "build", result.Stages[0].Name)
	assert.Equal(t, "running", result.Stages[0].Status)
}

func TestNewDrone(t *testing.T) {
	client := NewDrone("https://drone.example.com", "my-token")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}

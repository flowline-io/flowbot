package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
)

func TestListPipelineVersions(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   string
		seed       func(*testing.T, context.Context, *store.PipelineStore, *store.Client)
		wantStatus int
		wantBody   string
	}{
		{
			name:     "empty versions returns empty array",
			pipeline: "test-empty-vers",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, _ *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-empty-vers", ""))
			},
			wantStatus: http.StatusOK,
			wantBody:   "[]",
		},
		{
			name:     "returns version list after publish",
			pipeline: "test-has-vers",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-has-vers", ""))
				require.NoError(t, c.PipelineDefinition.Update().
					SetYamlDraft("name: tv\nsteps:\n  - name: s1").
					Where(pipelinedefinition.Name("test-has-vers")).
					Exec(ctx))
				_, err := s.PublishDefinition(ctx, "test-has-vers", 1)
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "version",
		},
		{
			name:       "pipeline not found returns 404",
			pipeline:   "no-such-pipeline",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			if tt.seed != nil {
				ps := store.NewPipelineStore(client)
				tt.seed(t, context.Background(), ps, client)
			}

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions", http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}

func TestGetPipelineVersion(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   string
		version    string
		seed       func(*testing.T, context.Context, *store.PipelineStore, *store.Client)
		wantStatus int
		wantBody   string
	}{
		{
			name:     "returns version YAML",
			pipeline: "test-get-vers",
			version:  "2",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-get-vers", ""))
				require.NoError(t, c.PipelineDefinition.Update().
					SetYamlDraft("name: test-get-vers\nsteps:\n  - name: s1").
					Where(pipelinedefinition.Name("test-get-vers")).
					Exec(ctx))
				_, err := s.PublishDefinition(ctx, "test-get-vers", 1)
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "yaml",
		},
		{
			name:     "version not found returns 404",
			pipeline: "test-get-nf",
			version:  "99",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, _ *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-get-nf", ""))
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "pipeline not found returns 404",
			pipeline:   "bad-pipe-99",
			version:    "1",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			if tt.seed != nil {
				ps := store.NewPipelineStore(client)
				tt.seed(t, context.Background(), ps, client)
			}

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions/"+tt.version, http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}

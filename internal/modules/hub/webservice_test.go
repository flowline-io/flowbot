package hub

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func mapDomainErrors(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return fiber.StatusBadRequest, true
	case errors.Is(err, types.ErrNotFound):
		return fiber.StatusNotFound, true
	default:
		return fiber.StatusInternalServerError, false
	}
}

func errorHandler(ctx fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	if code, ok := mapDomainErrors(err); ok {
		return ctx.Status(code).SendString(err.Error())
	}
	var e oops.OopsError
	if errors.As(err, &e) {
		if e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
			return ctx.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
}

func TestQueryByTag_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing key returns 400", "value=alpha", 400},
		{"missing value returns 400", "key=project", 400},
		{"empty key and value returns 400", "key=&value=", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/resource-chain", queryByTag)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/resource-chain?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestForgeWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 6 forge webservice rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, forgeWebserviceRules, 6)
			},
		},
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, forgeWebserviceRules)
			},
		},
		{
			name: "should have non-nil functions",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range forgeWebserviceRules {
					assert.NotNil(t, r.Function, "function should not be nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestGithubWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 9 github webservice rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, githubWebserviceRules, 9)
			},
		},
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, githubWebserviceRules)
			},
		},
		{
			name: "should have non-nil functions",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range githubWebserviceRules {
					assert.NotNil(t, r.Function, "function should not be nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestNoteWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, noteWebserviceRules)
			},
		},
		{
			name: "should contain CRUD endpoints",
			test: func(t *testing.T) {
				t.Parallel()
				paths := make(map[string]bool)
				for _, r := range noteWebserviceRules {
					paths[r.Path] = true
				}
				for _, expected := range []string{
					"/",
					"/:id",
					"/search",
					"/health",
					"/:id/content",
				} {
					assert.True(t, paths[expected], "expected path %q in note webservice rules", expected)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestFireflyiiiWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, fireflyiiiWebserviceRules)
			},
		},
		{
			name: "should contain finance endpoints",
			test: func(t *testing.T) {
				t.Parallel()
				paths := make(map[string]bool)
				for _, r := range fireflyiiiWebserviceRules {
					paths[r.Path] = true
				}
				for _, expected := range []string{
					"/transactions",
					"/about",
					"/user",
					"/health",
				} {
					assert.True(t, paths[expected], "expected path %q in fireflyiii webservice rules", expected)
				}
			},
		},
		{
			name: "should have four rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, fireflyiiiWebserviceRules, 4)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestTransmissionWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, transmissionWebserviceRules)
			},
		},
		{
			name: "should contain torrent endpoints",
			test: func(t *testing.T) {
				t.Parallel()
				paths := make(map[string]bool)
				for _, r := range transmissionWebserviceRules {
					paths[r.Path] = true
				}
				for _, expected := range []string{
					"/torrents",
					"/torrents/stop",
					"/torrents/remove",
					"/health",
				} {
					assert.True(t, paths[expected], "expected path %q in transmission webservice rules", expected)
				}
			},
		},
		{
			name: "should have five rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, transmissionWebserviceRules, 5)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestNocodbWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, nocodbWebserviceRules)
			},
		},
		{
			name: "should contain nocodb endpoints",
			test: func(t *testing.T) {
				t.Parallel()
				paths := make(map[string]bool)
				for _, r := range nocodbWebserviceRules {
					paths[r.Path] = true
				}
				for _, expected := range []string{
					"/bases",
					"/bases/:baseId/tables",
					"/tables/:tableId",
					"/tables/:tableId/records",
					"/tables/:tableId/records/:recordId",
					"/health",
				} {
					assert.True(t, paths[expected], "expected path %q in nocodb webservice rules", expected)
				}
			},
		},
		{
			name: "should have nine rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, nocodbWebserviceRules, 9)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestDevopsWebserviceRules_Structure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, devopsWebserviceRules)
			},
		},
		{
			name: "should contain devops endpoints",
			test: func(t *testing.T) {
				t.Parallel()
				paths := make(map[string]bool)
				for _, r := range devopsWebserviceRules {
					paths[r.Path] = true
				}
				for _, expected := range []string{
					"/status",
					"/beszel/systems",
					"/beszel/systems/:id",
					"/uptimekuma/health",
					"/uptimekuma/metrics",
					"/traefik/overview",
					"/traefik/routers",
					"/traefik/services",
					"/grafana/health",
					"/grafana/datasources",
					"/grafana/dashboards",
					"/grafana/query",
					"/wakapi/summary",
					"/wakapi/projects",
					"/dozzle/health",
					"/netalertx/health",
					"/netalertx/devices",
					"/netalertx/totals",
					"/netalertx/devices/search",
				} {
					assert.True(t, paths[expected], "expected path %q in devops webservice rules", expected)
				}
			},
		},
		{
			name: "should have nineteen rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, devopsWebserviceRules, 19)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestForgeGetRepo_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing owner returns 400", "repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg", 400},
		{"both missing returns 400", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/repo", forgeGetRepo)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/repo?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestForgeGetIssue_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing index returns 400", "owner=myorg&repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg&index=1", 400},
		{"missing owner returns 400", "repo=myrepo&index=1", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/issue", forgeGetIssue)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/issue?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestForgeGetCommitDiff_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing commit_id returns 400", "owner=myorg&repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg&commit_id=abc", 400},
		{"missing owner returns 400", "repo=myrepo&commit_id=abc", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/commit-diff", forgeGetCommitDiff)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/commit-diff?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestForgeGetFileContent_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing file_path returns 400", "owner=myorg&repo=myrepo&commit_id=abc", 400},
		{"missing commit_id returns 400", "owner=myorg&repo=myrepo&file_path=main.go", 400},
		{"missing owner returns 400", "repo=myrepo&commit_id=abc&file_path=main.go", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/file-content", forgeGetFileContent)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/file-content?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubGetRepo_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing owner returns 400", "repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg", 400},
		{"both missing returns 400", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/repo", githubGetRepo)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/repo?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubGetIssue_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing number returns 400", "owner=myorg&repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg&number=1", 400},
		{"missing owner returns 400", "repo=myrepo&number=1", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/issue", githubGetIssue)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/issue?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubGetCommitDiff_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing commit_id returns 400", "owner=myorg&repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg&commit_id=abc", 400},
		{"missing owner returns 400", "repo=myrepo&commit_id=abc", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/commit-diff", githubGetCommitDiff)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/commit-diff?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubGetFileContent_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing file_path returns 400", "owner=myorg&repo=myrepo&commit_id=abc", 400},
		{"missing commit_id returns 400", "owner=myorg&repo=myrepo&file_path=main.go", 400},
		{"missing owner returns 400", "repo=myrepo&commit_id=abc&file_path=main.go", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/file-content", githubGetFileContent)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/file-content?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubListReleases_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing owner returns 400", "repo=myrepo", 400},
		{"missing repo returns 400", "owner=myorg", 400},
		{"both missing returns 400", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/releases", githubListReleases)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/releases?"+tt.queryStr, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetRelations_Validation(t *testing.T) {
	tests := []struct {
		name       string
		app        string
		entityID   string
		wantStatus int
	}{
		{"empty app returns 400", "", "bm-123", 400},
		{"empty entity_id returns 400", "karakeep", "", 400},
		{"both empty returns 400", "", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/:app/:entity_id/relations", getRelations)
			defer app.Shutdown()
			// Build query params; use "_" sentinel for empty values since
			// Fiber returns "" for both unset and empty query params.
			qApp := tt.app
			if qApp == "" {
				qApp = "_"
			}
			qEntity := tt.entityID
			if qEntity == "" {
				qEntity = "_"
			}
			url := "/x/id/relations?app=" + qApp + "&entity_id=" + qEntity
			req := httptest.NewRequest(fiber.MethodGet, url, http.NoBody)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestWebserviceRules_Combined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should include all sub-module rules",
			test: func(t *testing.T) {
				t.Parallel()
				expectedMin := len(hubWebserviceRules) +
					len(bookmarkWebserviceRules) +
					len(kanbanWebserviceRules) +
					len(noteWebserviceRules) +
					len(readerWebserviceRules) +
					len(forgeWebserviceRules) +
					len(githubWebserviceRules)
				assert.GreaterOrEqual(t, len(webserviceRules), expectedMin)
			},
		},
		{
			name: "should be registered in Rules",
			test: func(t *testing.T) {
				t.Parallel()
				rules := handler.Rules()
				require.NotEmpty(t, rules)
				found := false
				for _, r := range rules {
					if ws, ok := r.([]webservice.Rule); ok && len(ws) > 0 {
						found = true
						break
					}
				}
				assert.True(t, found, "webservice rules should be in Rules()")
			},
		},
		{
			name: "should contain forge and github rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, forgeWebserviceRules)
				assert.NotEmpty(t, githubWebserviceRules)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

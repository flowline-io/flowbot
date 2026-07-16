package hub

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func registerTestInvoker(t *testing.T, capType hub.CapabilityType, op string, fn capability.Invoker) {
	t.Helper()
	require.NoError(t, capability.RegisterInvoker(capType, op, fn))
	t.Cleanup(func() {
		capability.UnregisterInvoker(capType, op)
	})
}

type relationQueryData struct {
	Downstream []any `json:"downstream"`
}

type relationQueryPayload struct {
	Data relationQueryData `json:"data"`
}

func TestQueryOrParam(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "prefers query value", url: "/x/y?app=karakeep", want: "karakeep"},
		{name: "underscore sentinel returns empty", url: "/fallback/y?app=_", want: ""},
		{name: "empty query uses path param", url: "/entity-1/y", want: "entity-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Get("/:app/:entity_id", func(c fiber.Ctx) error {
				return c.SendString(queryOrParam(c, "app"))
			})
			req := httptest.NewRequest(fiber.MethodGet, tt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(body))
		})
	}
}

func TestPageParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		query string
		check func(t *testing.T, got map[string]any)
	}{
		{
			name:  "empty query returns empty map",
			query: "",
			check: func(t *testing.T, got map[string]any) {
				t.Helper()
				assert.Empty(t, got)
			},
		},
		{
			name:  "valid limit parsed",
			query: "limit=25",
			check: func(t *testing.T, got map[string]any) {
				t.Helper()
				assert.InDelta(t, 25, got["limit"], 0.001)
			},
		},
		{
			name:  "invalid limit ignored",
			query: "limit=abc",
			check: func(t *testing.T, got map[string]any) {
				t.Helper()
				assert.Empty(t, got)
			},
		},
		{
			name:  "cursor preserved",
			query: "cursor=next-page",
			check: func(t *testing.T, got map[string]any) {
				t.Helper()
				assert.Equal(t, "next-page", got["cursor"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Get("/", func(c fiber.Ctx) error {
				params := pageParams(c)
				return c.JSON(params)
			})
			req := httptest.NewRequest(fiber.MethodGet, "/?"+tt.query, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&got))
			tt.check(t, got)
		})
	}
}

func TestCheckURLExists_Validation(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{name: "missing url returns 400", query: "", wantStatus: 400},
		{name: "empty url returns 400", query: "url=", wantStatus: 400},
		{name: "valid url invokes capability", query: "url=https://example.com", wantStatus: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantStatus == 200 {
				registerTestInvoker(t, hub.CapKarakeep, capability.OpBookmarkCheckURL, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					return &capability.InvokeResult{Data: map[string]any{"exists": true}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Get("/check-url", checkURLExists)
			req := httptest.NewRequest(fiber.MethodGet, "/check-url?"+tt.query, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestListBookmarks_InvokeSuccess(t *testing.T) {
	t.Parallel()
	registerTestInvoker(t, hub.CapKarakeep, capability.OpBookmarkList, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"items": []any{}, "params": params}}, nil
	})

	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/", listBookmarks)
	req := httptest.NewRequest(fiber.MethodGet, "/?limit=10&archived=true", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"ok"`)
}

func TestListFeeds_InvokeSuccess(t *testing.T) {
	t.Parallel()
	registerTestInvoker(t, hub.CapMiniflux, capability.OpReaderListFeeds, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"feeds": []any{}}}, nil
	})

	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/feeds", listFeeds)
	req := httptest.NewRequest(fiber.MethodGet, "/feeds", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetRelations_Success(t *testing.T) {
	tests := []struct {
		name       string
		seed       func(context.Context, *store.Client) error
		appName    string
		entityID   string
		wantStatus int
		wantDown   int
	}{
		{
			name:       "empty relations returns scaffold",
			appName:    "karakeep",
			entityID:   "bm-1",
			wantStatus: 200,
		},
		{
			name: "returns downstream relation",
			seed: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-1").
					SetTargetEventID("tgt-1").
					SetSourceApp("karakeep").
					SetSourceCapability("bookmark").
					SetSourceEntityID("bm-1").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("issue-9").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			appName:    "karakeep",
			entityID:   "bm-1",
			wantStatus: 200,
			wantDown:   1,
		},
		{
			name:       "missing app still validated via query sentinel",
			appName:    "",
			entityID:   "bm-1",
			wantStatus: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqlitetest.OpenClient(t, tt.name)
			oldStore := rcStore
			rcStore = store.NewResourceChainStore(client)
			t.Cleanup(func() { rcStore = oldStore })

			if tt.seed != nil {
				require.NoError(t, tt.seed(context.Background(), client))
			}

			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Get("/:app/:entity_id/relations", getRelations)

			qApp := tt.appName
			if qApp == "" {
				qApp = "_"
			}
			req := httptest.NewRequest(fiber.MethodGet, "/x/"+tt.entityID+"/relations?app="+qApp+"&entity_id="+tt.entityID, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantDown > 0 {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				var payload relationQueryPayload
				require.NoError(t, sonic.Unmarshal(body, &payload))
				assert.Len(t, payload.Data.Downstream, tt.wantDown)
			}
		})
	}
}

func TestCreateBookmark_Validation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid json returns 400", body: `{`, wantStatus: 400},
		{name: "missing url returns 400", body: `{}`, wantStatus: 400},
		{name: "invalid url returns 400", body: `{"url":"not-a-url"}`, wantStatus: 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Post("/", createBookmark)
			req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestInvokeBookmark_PropagatesError(t *testing.T) {
	t.Parallel()
	registerTestInvoker(t, hub.CapKarakeep, capability.OpBookmarkGet, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return nil, types.Errorf(types.ErrNotFound, "bookmark missing")
	})

	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/:id", getBookmark)
	req := httptest.NewRequest(fiber.MethodGet, "/missing-id", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestListTasks_InvokeSuccess(t *testing.T) {
	registerTestInvoker(t, hub.CapKanboard, capability.OpKanbanListTasks, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: []any{map[string]any{"id": 1, "title": "Task"}}, Meta: params}, nil
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/", listTasks)
	req := httptest.NewRequest(fiber.MethodGet, "/?project_id=1&status_id=1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTasks_InvokeError(t *testing.T) {
	registerTestInvoker(t, hub.CapKanboard, capability.OpKanbanListTasks, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return nil, types.Errorf(types.ErrProvider, "kanban down")
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/", listTasks)
	req := httptest.NewRequest(fiber.MethodGet, "/?project_id=1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestGetTask_ValidationAndSuccess(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		register   bool
		invokeErr  bool
		wantStatus int
	}{
		{name: "invalid id returns 400", path: "/abc", wantStatus: 400},
		{name: "get task success", path: "/42", register: true, wantStatus: 200},
		{name: "get task invoke error", path: "/42", register: true, invokeErr: true, wantStatus: 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register {
				registerTestInvoker(t, hub.CapKanboard, capability.OpKanbanGetTask, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					if tt.invokeErr {
						return nil, types.Errorf(types.ErrNotFound, "missing task")
					}
					return &capability.InvokeResult{Data: map[string]any{"id": 42, "title": "Fix"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Get("/:id", getTask)
			req := httptest.NewRequest(fiber.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestCreateTask_ValidationAndSuccess(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		register   bool
		wantStatus int
	}{
		{name: "invalid json returns 400", body: `{`, wantStatus: 400},
		{name: "missing title returns 400", body: `{}`, wantStatus: 400},
		{name: "create task success", body: `{"title":"New task","project_id":1}`, register: true, wantStatus: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register {
				registerTestInvoker(t, hub.CapKanboard, capability.OpKanbanCreateTask, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					return &capability.InvokeResult{Data: map[string]any{"id": 99}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Post("/", createTask)
			req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestListNotes_InvokeSuccess(t *testing.T) {
	registerTestInvoker(t, hub.CapTrilium, capability.OpNoteList, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"items": []any{}}}, nil
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/", listNotes)
	req := httptest.NewRequest(fiber.MethodGet, "/?limit=10", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateNote_ValidationAndSuccess(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		register   bool
		invokeErr  bool
		wantStatus int
	}{
		{name: "missing title returns 400", body: `{"content":"body"}`, wantStatus: 400},
		{name: "create note success", body: `{"title":"Note","content":"body"}`, register: true, wantStatus: 200},
		{name: "create note invoke error", body: `{"title":"Note"}`, register: true, invokeErr: true, wantStatus: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register {
				registerTestInvoker(t, hub.CapTrilium, capability.OpNoteCreate, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					if tt.invokeErr {
						return nil, types.Errorf(types.ErrProvider, "note create failed")
					}
					return &capability.InvokeResult{Data: map[string]any{"id": "n-1"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Post("/", createNote)
			req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestSearchNotes_MissingQuery(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/search", searchNotes)
	req := httptest.NewRequest(fiber.MethodGet, "/search", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestListMemos_InvokeSuccess(t *testing.T) {
	registerTestInvoker(t, hub.CapMemos, capability.OpMemoList, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"items": []any{}}, Page: &capability.PageInfo{}}, nil
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/", listMemos)
	req := httptest.NewRequest(fiber.MethodGet, "/?limit=5", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateMemo_ValidationAndSuccess(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		register   bool
		invokeErr  bool
		wantStatus int
	}{
		{name: "missing content returns 400", body: `{}`, wantStatus: 400},
		{name: "create memo success", body: `{"content":"hello"}`, register: true, wantStatus: 200},
		{name: "create memo invoke error", body: `{"content":"hello"}`, register: true, invokeErr: true, wantStatus: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register {
				registerTestInvoker(t, hub.CapMemos, capability.OpMemoCreate, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					if tt.invokeErr {
						return nil, types.Errorf(types.ErrProvider, "memo create failed")
					}
					return &capability.InvokeResult{Data: map[string]any{"name": "memos/1"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Post("/", createMemo)
			req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestDeleteMemo_MissingName(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Delete("/", deleteMemo)
	req := httptest.NewRequest(fiber.MethodDelete, "/", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestForgeHandlers_InvokePaths(t *testing.T) {
	tests := []struct {
		name       string
		registerOp string
		url        string
		invokeErr  bool
		wantStatus int
	}{
		{
			name:       "forge get user success",
			registerOp: capability.OpForgeGetUser,
			url:        "/user",
			wantStatus: 200,
		},
		{
			name:       "forge get user error",
			registerOp: capability.OpForgeGetUser,
			url:        "/user",
			invokeErr:  true,
			wantStatus: 500,
		},
		{
			name:       "forge get repo missing params",
			url:        "/repo",
			wantStatus: 400,
		},
		{
			name:       "forge get repo success",
			registerOp: capability.OpForgeGetRepo,
			url:        "/repo?owner=o&repo=r",
			wantStatus: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.registerOp != "" {
				registerTestInvoker(t, hub.CapGitea, tt.registerOp, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					if tt.invokeErr {
						return nil, types.Errorf(types.ErrProvider, "forge failed")
					}
					return &capability.InvokeResult{Data: map[string]any{"username": "dev"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Get("/user", forgeGetUser)
			app.Get("/repo", forgeGetRepo)
			req := httptest.NewRequest(fiber.MethodGet, tt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGithubHandlers_InvokePaths(t *testing.T) {
	tests := []struct {
		name       string
		registerOp string
		url        string
		invokeErr  bool
		wantStatus int
	}{
		{name: "github get user success", registerOp: capability.OpGithubGetUser, url: "/user", wantStatus: 200},
		{name: "github get user error", registerOp: capability.OpGithubGetUser, url: "/user", invokeErr: true, wantStatus: 500},
		{name: "github get repo missing params", url: "/repo", wantStatus: 400},
		{name: "github get repo success", registerOp: capability.OpGithubGetRepo, url: "/repo?owner=o&repo=r", wantStatus: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.registerOp != "" {
				registerTestInvoker(t, hub.CapGithub, tt.registerOp, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
					if tt.invokeErr {
						return nil, types.Errorf(types.ErrProvider, "github failed")
					}
					return &capability.InvokeResult{Data: map[string]any{"login": "octo"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Get("/user", githubGetUser)
			app.Get("/repo", githubGetRepo)
			req := httptest.NewRequest(fiber.MethodGet, tt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestSearchBookmarks_InvokeSuccess(t *testing.T) {
	registerTestInvoker(t, hub.CapKarakeep, capability.OpBookmarkSearch, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"items": []any{}, "query": params["q"]}}, nil
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/search", searchBookmarks)
	req := httptest.NewRequest(fiber.MethodGet, "/search?q=go", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestArchiveBookmark_InvokeSuccess(t *testing.T) {
	registerTestInvoker(t, hub.CapKarakeep, capability.OpBookmarkArchive, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"archived": true}}, nil
	})
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Patch("/:id", archiveBookmark)
	req := httptest.NewRequest(fiber.MethodPatch, "/bm-1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

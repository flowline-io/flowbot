package note

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// Config holds mock backend behavior for conformance testing each Service method.
type Config struct {
	ListResult    *ability.ListResult[ability.Note]
	ListErr       error
	GetItem       *ability.Note
	GetErr        error
	CreateItem    *ability.Note
	CreateErr     error
	UpdateItem    *ability.Note
	UpdateErr     error
	DeleteErr     error
	Content       string
	ContentErr    error
	SetContentErr error
	SearchResult  *ability.ListResult[ability.Note]
	SearchErr     error
	AppInfo       *ability.Note
	AppInfoErr    error
}

// ServiceFactory creates a Service from a Config for conformance testing.
type ServiceFactory func(t *testing.T, cfg Config) Service

// RunNoteConformance runs the full note capability conformance test suite.
//
//revive:disable:cyclomatic — conformance suites test every operation with multiple scenarios.
func RunNoteConformance(t *testing.T, factory ServiceFactory) {
	t.Helper()

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantLen int
			wantErr bool
		}{
			{
				name: "success with items",
				cfg: Config{ListResult: &ability.ListResult[ability.Note]{
					Items: []*ability.Note{{ID: "n-1"}, {ID: "n-2"}},
				}},
				wantLen: 2,
				wantErr: false,
			},
			{
				name:    "empty list",
				cfg:     Config{ListResult: &ability.ListResult[ability.Note]{Items: []*ability.Note{}}},
				wantLen: 0,
				wantErr: false,
			},
			{
				name:    "provider error",
				cfg:     Config{ListErr: assert.AnError},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				result, err := svc.List(context.Background(), &ListQuery{})
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Items, tt.wantLen)
			})
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{
				name:    "success",
				cfg:     Config{GetItem: &ability.Note{ID: "n-1", Title: "test"}},
				id:      "n-1",
				wantErr: false,
			},
			{
				name:    "empty id returns error",
				cfg:     Config{},
				id:      "",
				wantErr: true,
			},
			{
				name:    "provider error",
				cfg:     Config{GetErr: assert.AnError},
				id:      "n-1",
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.Get(context.Background(), tt.id)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			title   string
			wantErr bool
		}{
			{name: "success", cfg: Config{CreateItem: &ability.Note{ID: "new", Title: "test"}}, title: "test", wantErr: false},
			{name: "empty title returns error", cfg: Config{}, title: "", wantErr: true},
			{name: "provider error", cfg: Config{CreateErr: assert.AnError}, title: "test", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.Create(context.Background(), tt.title, "", "text", "")
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{name: "success", cfg: Config{UpdateItem: &ability.Note{ID: "n-1", Title: "updated"}}, id: "n-1", wantErr: false},
			{name: "empty id returns error", cfg: Config{}, id: "", wantErr: true},
			{name: "provider error", cfg: Config{UpdateErr: assert.AnError}, id: "n-1", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.Update(context.Background(), tt.id, "new title", "")
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{name: "success", cfg: Config{}, id: "n-1", wantErr: false},
			{name: "empty id returns error", cfg: Config{}, id: "", wantErr: true},
			{name: "provider error", cfg: Config{DeleteErr: assert.AnError}, id: "n-1", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				err := svc.Delete(context.Background(), tt.id)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("GetContent", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			want    string
			wantErr bool
		}{
			{name: "success", cfg: Config{Content: "hello world"}, id: "n-1", want: "hello world", wantErr: false},
			{name: "empty id returns error", cfg: Config{}, id: "", wantErr: true},
			{name: "provider error", cfg: Config{ContentErr: assert.AnError}, id: "n-1", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				content, err := svc.GetContent(context.Background(), tt.id)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, content)
			})
		}
	})

	t.Run("SetContent", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{name: "success", cfg: Config{}, id: "n-1", wantErr: false},
			{name: "empty id returns error", cfg: Config{}, id: "", wantErr: true},
			{name: "provider error", cfg: Config{SetContentErr: assert.AnError}, id: "n-1", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				err := svc.SetContent(context.Background(), tt.id, "new content")
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("Search", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantLen int
			wantErr bool
		}{
			{
				name: "success with results",
				cfg: Config{SearchResult: &ability.ListResult[ability.Note]{
					Items: []*ability.Note{{ID: "n-1", Title: "Match"}},
				}},
				wantLen: 1,
				wantErr: false,
			},
			{
				name:    "empty results",
				cfg:     Config{SearchResult: &ability.ListResult[ability.Note]{Items: []*ability.Note{}}},
				wantLen: 0,
				wantErr: false,
			},
			{
				name:    "provider error",
				cfg:     Config{SearchErr: assert.AnError},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				result, err := svc.Search(context.Background(), "test")
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result.Items, tt.wantLen)
			})
		}
	})

	t.Run("GetAppInfo", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantErr bool
		}{
			{name: "success", cfg: Config{AppInfo: &ability.Note{ID: "instance", Title: "Trilium"}}, wantErr: false},
			{name: "provider error", cfg: Config{AppInfoErr: assert.AnError}, wantErr: true},
			{name: "success with empty instance", cfg: Config{AppInfo: &ability.Note{}}, wantErr: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				info, err := svc.GetAppInfo(context.Background())
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, info)
			})
		}
	})
}

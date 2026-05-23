package example

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// Config holds mock backend behavior for conformance testing each Service method.
type Config struct {
	ListItems  []*ability.Host
	ListErr    error
	GetItem    *ability.Host
	GetErr     error
	CreateItem *ability.Host
	CreateErr  error
	UpdateItem *ability.Host
	UpdateErr  error
	DeleteErr  error
	HealthOk   bool
	HealthErr  error
	RawItems   []any
	RawCursor  string
	RawErr     error
}

// ServiceFactory creates a Service from a Config for conformance testing.
type ServiceFactory func(t *testing.T, cfg Config) Service

// RunExampleConformance runs the full example capability conformance test suite.
func RunExampleConformance(t *testing.T, factory ServiceFactory) {
	t.Helper()

	t.Run("GetItem", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{
				name:    "success",
				cfg:     Config{GetItem: &ability.Host{ID: "h-1", Name: "test"}},
				id:      "h-1",
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
				id:      "h-1",
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.GetItem(context.Background(), tt.id)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("ListItems", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantLen int
			wantErr bool
		}{
			{
				name:    "success with items",
				cfg:     Config{ListItems: []*ability.Host{{ID: "h-1"}, {ID: "h-2"}}},
				wantLen: 2,
				wantErr: false,
			},
			{
				name:    "empty list",
				cfg:     Config{ListItems: []*ability.Host{}},
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
				result, err := svc.ListItems(context.Background(), &ListQuery{})
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

	t.Run("CreateItem", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			title   string
			wantErr bool
		}{
			{name: "success", cfg: Config{CreateItem: &ability.Host{ID: "new", Name: "test"}}, title: "test", wantErr: false},
			{name: "empty title returns error", cfg: Config{}, title: "", wantErr: true},
			{name: "provider error", cfg: Config{CreateErr: assert.AnError}, title: "test", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.CreateItem(context.Background(), tt.title, nil)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantOk  bool
			wantErr bool
		}{
			{name: "healthy", cfg: Config{HealthOk: true}, wantOk: true, wantErr: false},
			{name: "unhealthy", cfg: Config{HealthOk: false}, wantOk: false, wantErr: false},
			{name: "error", cfg: Config{HealthErr: assert.AnError}, wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				ok, err := svc.HealthCheck(context.Background())
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.wantOk, ok)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()
		t.Run("canceled context", func(t *testing.T) {
			t.Parallel()
			svc := factory(t, Config{GetItem: &ability.Host{ID: "h-1"}})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := svc.GetItem(ctx, "h-1")
			require.Error(t, err)
		})
	})
}

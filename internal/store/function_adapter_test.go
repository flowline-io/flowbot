package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestFunctionCatalogAdapter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, catalog functions.Catalog)
	}{
		{
			name: "create publish get delete",
			run: func(t *testing.T, catalog functions.Catalog) {
				ctx := context.Background()
				meta := "name: adapter-fn\nhttp:\n  auth:\n    token: t\n"
				require.NoError(t, catalog.Create(ctx, "adapter-fn", meta, "main.py", "print(1)", "uid-1"))

				def, err := catalog.GetByName(ctx, "adapter-fn")
				require.NoError(t, err)
				assert.Equal(t, 1, def.Version)
				assert.Equal(t, string(types.FunctionDefinitionDraft), def.Status)

				published, err := catalog.Publish(ctx, "adapter-fn", 1)
				require.NoError(t, err)
				assert.Equal(t, string(types.FunctionDefinitionPublished), published.Status)
				assert.Equal(t, 2, published.Version)

				ver, err := catalog.GetLatestPublished(ctx, "adapter-fn")
				require.NoError(t, err)
				assert.Equal(t, 1, ver.Version)
				assert.Equal(t, "main.py", ver.Entrypoint)

				n, err := catalog.Delete(ctx, "adapter-fn")
				require.NoError(t, err)
				assert.GreaterOrEqual(t, n, int64(0))
				_, err = catalog.GetByName(ctx, "adapter-fn")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "idempotent create run conflict",
			run: func(t *testing.T, catalog functions.Catalog) {
				ctx := context.Background()
				meta := "name: idem-fn\nhttp:\n  auth:\n    token: t\n"
				require.NoError(t, catalog.Create(ctx, "idem-fn", meta, "main.py", "print(1)", "uid-1"))
				_, err := catalog.Publish(ctx, "idem-fn", 1)
				require.NoError(t, err)

				run, err := catalog.CreateRun(ctx, "idem-fn", 1, nil)
				require.NoError(t, err)
				assert.Equal(t, string(types.FunctionRunRunning), run.Status)

				key := "idem-1"
				_, err = catalog.CreateRun(ctx, "idem-fn", 1, &key)
				require.NoError(t, err)
				_, err = catalog.CreateRun(ctx, "idem-fn", 1, &key)
				require.ErrorIs(t, err, types.ErrConflict)
			},
		},
		{
			name: "complete run and list",
			run: func(t *testing.T, catalog functions.Catalog) {
				ctx := context.Background()
				meta := "name: complete-fn\nhttp:\n  auth:\n    token: t\n"
				require.NoError(t, catalog.Create(ctx, "complete-fn", meta, "main.py", "print(1)", "uid-1"))
				_, err := catalog.Publish(ctx, "complete-fn", 1)
				require.NoError(t, err)
				run, err := catalog.CreateRun(ctx, "complete-fn", 1, nil)
				require.NoError(t, err)
				result := `{"ok":true}`
				exit := 0
				done, err := catalog.CompleteRun(ctx, run.ID, string(types.FunctionRunSucceeded), 12, &exit, "", &result)
				require.NoError(t, err)
				assert.Equal(t, string(types.FunctionRunSucceeded), done.Status)
				runs, err := catalog.ListRuns(ctx, "complete-fn")
				require.NoError(t, err)
				require.Len(t, runs, 1)
				assert.Equal(t, run.ID, runs[0].ID)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := store.NewFunctionStore(sqlitetest.OpenClient(t, t.Name()))
			tt.run(t, store.NewFunctionCatalogAdapter(fs))
		})
	}
}

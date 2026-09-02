package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestFunctionStore_DefinitionCRUD(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, fs *store.FunctionStore)
	}{
		{
			name: "create get and update draft bumps version",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "parse-bill", "user-a"))

				def, err := fs.GetDefinitionByName(ctx, "parse-bill")
				require.NoError(t, err)
				assert.Equal(t, "parse-bill", def.Name)
				assert.Equal(t, "draft", string(def.Status))
				assert.Equal(t, 1, def.Version)
				assert.Equal(t, "user-a", def.CreatedBy)

				def, err = fs.UpdateDefinitionDraft(ctx, "parse-bill", "name: parse-bill\n", "main.py", "print(1)", 1)
				require.NoError(t, err)
				assert.Equal(t, 2, def.Version)
				assert.Equal(t, "main.py", def.EntrypointDraft)
				assert.Equal(t, "print(1)", def.SourceDraft)
			},
		},
		{
			name: "update draft with stale version returns ErrConflict",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "stale-fn", ""))
				_, err := fs.UpdateDefinitionDraft(ctx, "stale-fn", "m", "main.py", "x", 1)
				require.NoError(t, err)
				_, err = fs.UpdateDefinitionDraft(ctx, "stale-fn", "m2", "main.py", "y", 1)
				require.ErrorIs(t, err, types.ErrConflict)
			},
		},
		{
			name: "duplicate create returns ErrAlreadyExists",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "dup-fn", ""))
				err := fs.CreateDefinition(ctx, "dup-fn", "")
				require.ErrorIs(t, err, types.ErrAlreadyExists)
			},
		},
		{
			name: "get missing returns ErrNotFound",
			run: func(t *testing.T, fs *store.FunctionStore) {
				_, err := fs.GetDefinitionByName(context.Background(), "missing-fn")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "delete removes definition versions and runs",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "del-fn", ""))
				_, err := fs.UpdateDefinitionDraft(ctx, "del-fn", "meta", "main.py", "print(1)", 1)
				require.NoError(t, err)
				_, err = fs.PublishDefinition(ctx, "del-fn", 2)
				require.NoError(t, err)
				_, err = fs.CreateRun(ctx, "del-fn", 1, "running", "")
				require.NoError(t, err)

				_, err = fs.DeleteDefinitionByName(ctx, "del-fn")
				require.NoError(t, err)
				_, err = fs.GetDefinitionByName(ctx, "del-fn")
				require.ErrorIs(t, err, types.ErrNotFound)
				_, err = fs.GetLatestPublished(ctx, "del-fn")
				require.ErrorIs(t, err, types.ErrNotFound)
				runs, err := fs.ListRunsByName(ctx, "del-fn")
				require.NoError(t, err)
				assert.Empty(t, runs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, store.NewFunctionStore(sqlitetest.OpenClient(t, t.Name())))
		})
	}
}

func TestFunctionStore_PublishAndVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, fs *store.FunctionStore)
	}{
		{
			name: "publish copies draft and inserts version snapshot",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "pub-fn", ""))
				_, err := fs.UpdateDefinitionDraft(ctx, "pub-fn", "name: pub-fn\n", "main.py", "print(1)", 1)
				require.NoError(t, err)

				def, err := fs.PublishDefinition(ctx, "pub-fn", 2)
				require.NoError(t, err)
				assert.Equal(t, "published", string(def.Status))
				assert.Equal(t, 3, def.Version)
				require.NotNil(t, def.EntrypointPublished)
				assert.Equal(t, "main.py", *def.EntrypointPublished)
				require.NotNil(t, def.SourcePublished)
				assert.Equal(t, "print(1)", *def.SourcePublished)

				ver, err := fs.GetPublishedVersion(ctx, "pub-fn", 1)
				require.NoError(t, err)
				assert.Equal(t, "main.py", ver.Entrypoint)
				assert.Equal(t, "print(1)", ver.Source)

				latest, err := fs.GetLatestPublished(ctx, "pub-fn")
				require.NoError(t, err)
				assert.Equal(t, 1, latest.Version)

				listed, err := fs.ListPublishedDefinitions(ctx)
				require.NoError(t, err)
				require.Len(t, listed, 1)
				assert.Equal(t, "pub-fn", listed[0].Name)
			},
		},
		{
			name: "publish empty draft returns ErrConflict",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "empty-pub", ""))
				_, err := fs.PublishDefinition(ctx, "empty-pub", 1)
				require.ErrorIs(t, err, types.ErrConflict)
			},
		},
		{
			name: "publish stale version returns ErrConflict",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "stale-pub", ""))
				_, err := fs.UpdateDefinitionDraft(ctx, "stale-pub", "m", "main.py", "x", 1)
				require.NoError(t, err)
				_, err = fs.PublishDefinition(ctx, "stale-pub", 1)
				require.ErrorIs(t, err, types.ErrConflict)
			},
		},
		{
			name: "GetPublishedVersion missing returns ErrNotFound",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "ver-nf", ""))
				_, err := fs.GetPublishedVersion(ctx, "ver-nf", 99)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "republish stores newer latest version",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				require.NoError(t, fs.CreateDefinition(ctx, "repub-fn", ""))
				_, err := fs.UpdateDefinitionDraft(ctx, "repub-fn", "m1", "main.py", "v1", 1)
				require.NoError(t, err)
				_, err = fs.PublishDefinition(ctx, "repub-fn", 2)
				require.NoError(t, err)

				_, err = fs.UpdateDefinitionDraft(ctx, "repub-fn", "m2", "main.sh", "v2", 3)
				require.NoError(t, err)
				_, err = fs.PublishDefinition(ctx, "repub-fn", 4)
				require.NoError(t, err)

				latest, err := fs.GetLatestPublished(ctx, "repub-fn")
				require.NoError(t, err)
				assert.Equal(t, 2, latest.Version)
				assert.Equal(t, "main.sh", latest.Entrypoint)
				assert.Equal(t, "v2", latest.Source)

				old, err := fs.GetPublishedVersion(ctx, "repub-fn", 1)
				require.NoError(t, err)
				assert.Equal(t, "v1", old.Source)

				versions, err := fs.ListPublishedVersionNumbers(ctx, "repub-fn")
				require.NoError(t, err)
				assert.Equal(t, []int{2, 1}, versions)

				byName, err := fs.ListPublishedVersionNumbersByNames(ctx, []string{"repub-fn", "missing-fn"})
				require.NoError(t, err)
				assert.Equal(t, []int{2, 1}, byName["repub-fn"])
				assert.Empty(t, byName["missing-fn"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, store.NewFunctionStore(sqlitetest.OpenClient(t, t.Name())))
		})
	}
}

func TestFunctionStore_Runs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, fs *store.FunctionStore)
	}{
		{
			name: "create update and list runs",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				run, err := fs.CreateRun(ctx, "run-fn", 2, "running", "")
				require.NoError(t, err)
				require.NotNil(t, run)
				assert.Equal(t, "running", string(run.Status))
				assert.Nil(t, run.IdempotencyKey)

				code := 0
				result := `{"ok":true}`
				updated, err := fs.UpdateRun(ctx, run.ID, "succeeded", 12, &code, "", &result)
				require.NoError(t, err)
				assert.Equal(t, "succeeded", string(updated.Status))
				assert.Equal(t, int64(12), updated.DurationMs)
				require.NotNil(t, updated.ExitCode)
				assert.Equal(t, 0, *updated.ExitCode)
				require.NotNil(t, updated.ResultJSON)
				assert.Equal(t, result, *updated.ResultJSON)

				runs, err := fs.ListRunsByName(ctx, "run-fn")
				require.NoError(t, err)
				require.Len(t, runs, 1)
			},
		},
		{
			name: "idempotency key unique and lookup",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				run, err := fs.CreateRun(ctx, "idem-fn", 1, "running", "key-1")
				require.NoError(t, err)
				require.NotNil(t, run.IdempotencyKey)
				assert.Equal(t, "key-1", *run.IdempotencyKey)

				_, err = fs.CreateRun(ctx, "idem-fn", 1, "running", "key-1")
				require.ErrorIs(t, err, types.ErrConflict)

				got, err := fs.GetRunByIdempotencyKey(ctx, "idem-fn", "key-1")
				require.NoError(t, err)
				assert.Equal(t, run.ID, got.ID)
			},
		},
		{
			name: "empty idempotency keys do not collide",
			run: func(t *testing.T, fs *store.FunctionStore) {
				ctx := context.Background()
				_, err := fs.CreateRun(ctx, "empty-key-fn", 1, "running", "")
				require.NoError(t, err)
				_, err = fs.CreateRun(ctx, "empty-key-fn", 1, "running", "   ")
				require.NoError(t, err)

				runs, err := fs.ListRunsByName(ctx, "empty-key-fn")
				require.NoError(t, err)
				require.Len(t, runs, 2)

				_, err = fs.GetRunByIdempotencyKey(ctx, "empty-key-fn", "")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "update missing run returns ErrNotFound",
			run: func(t *testing.T, fs *store.FunctionStore) {
				_, err := fs.UpdateRun(context.Background(), 99999, "failed", 0, nil, "boom", nil)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "get missing idempotency key returns ErrNotFound",
			run: func(t *testing.T, fs *store.FunctionStore) {
				_, err := fs.GetRunByIdempotencyKey(context.Background(), "no-fn", "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, store.NewFunctionStore(sqlitetest.OpenClient(t, t.Name())))
		})
	}
}

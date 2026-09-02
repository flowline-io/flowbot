package functions_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeRunExitCodeAndStdoutJSON(t *testing.T) {
	t.Parallel()
	cat := newFakeCatalog()
	seedPublished(t, cat, "exit-fn", "main.py", "ignored")
	ver := 1

	exitSvc := functions.NewService(cat, &fakeExecProvider{exit: 2, stderr: "can't open file"})
	exitSvc.SetChecker(dcg.AllowAllChecker{})
	_, err := exitSvc.Invoke(context.Background(), functions.InvokeRequest{
		Name:    "exit-fn",
		Version: &ver,
		Event:   map[string]any{},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvokeRun)
	assert.Contains(t, err.Error(), "exited with code 2")
	assert.Contains(t, err.Error(), "can't open file")

	jsonSvc := functions.NewService(cat, &fakeExecProvider{stdout: "not-json", exit: 0})
	jsonSvc.SetChecker(dcg.AllowAllChecker{})
	_, err = jsonSvc.Invoke(context.Background(), functions.InvokeRequest{
		Name:    "exit-fn",
		Version: &ver,
		Event:   map[string]any{},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvokeRun)
	assert.Contains(t, err.Error(), "valid JSON")
}

func TestParseStdoutJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		stdout  string
		wantErr string
		want    any
	}{
		{name: "object", stdout: ` {"a":1} `, want: map[string]any{"a": float64(1)}},
		{name: "empty", stdout: "  ", wantErr: "empty"},
		{name: "two values", stdout: "1 2", wantErr: "exactly one"},
		{name: "too large", stdout: string(make([]byte, functions.MaxJSONBytes+1)), wantErr: "exceeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := functions.ParseStdoutJSON(tt.stdout)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateListAllDraftPublish(t *testing.T) {
	t.Parallel()
	cat := newFakeCatalog()
	svc := functions.NewService(cat, &fakeExecProvider{stdout: `{}`})
	svc.SetChecker(dcg.AllowAllChecker{})

	draft, err := svc.Create(context.Background(), "ui-fn", "main.py", "tester")
	require.NoError(t, err)
	require.NotNil(t, draft)
	assert.Equal(t, "ui-fn", draft.Name)
	assert.Equal(t, string(types.FunctionDefinitionDraft), draft.Status)
	assert.True(t, draft.TokenSet)
	assert.Equal(t, "••••••••", draft.Token)
	assert.NotEmpty(t, draft.Source)
	assert.Nil(t, draft.PublishedVersion)

	all, err := svc.ListAll(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "ui-fn", all[0].Name)
	assert.Equal(t, string(types.FunctionDefinitionDraft), all[0].Status)
	assert.Nil(t, all[0].PublishedVersion)

	pubOnly, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pubOnly)

	meta := "name: ui-fn\nhttp:\n  auth:\n    token: " + "••••••••" + "\nenv:\n  mode: test\n"
	saved, err := svc.SaveDraft(context.Background(), "ui-fn", meta, "main.py", "print('{\"ok\":true}')\n", draft.Version)
	require.NoError(t, err)
	assert.Equal(t, draft.Version+1, saved.Version)
	assert.Equal(t, "test", saved.Env["mode"])
	assert.True(t, saved.TokenSet)

	_, err = svc.SaveDraft(context.Background(), "ui-fn", meta, "main.py", "print(1)\n", draft.Version)
	require.ErrorIs(t, err, types.ErrConflict)

	published, err := svc.Publish(context.Background(), "ui-fn", saved.Version)
	require.NoError(t, err)
	assert.Equal(t, string(types.FunctionDefinitionPublished), published.Status)
	require.NotNil(t, published.Version)

	after, err := svc.GetDraft(context.Background(), "ui-fn")
	require.NoError(t, err)
	require.NotNil(t, after.PublishedVersion)
	assert.False(t, after.HasUnpublishedChanges)

	metaDirty := "name: ui-fn\nhttp:\n  auth:\n    token: " + "••••••••" + "\nenv:\n  mode: dirty\n"
	dirty, err := svc.SaveDraft(context.Background(), "ui-fn", metaDirty, "main.py", after.Source, after.Version)
	require.NoError(t, err)
	assert.True(t, dirty.HasUnpublishedChanges)

	ver := *after.PublishedVersion
	got, err := svc.Invoke(context.Background(), functions.InvokeRequest{
		Name:    "ui-fn",
		Version: &ver,
		Event:   map[string]any{},
	})
	require.NoError(t, err)
	assert.Equal(t, ver, got.Version)
}

func TestSaveDraftSecretRotateAndClear(t *testing.T) {
	t.Parallel()
	cat := newFakeCatalog()
	svc := functions.NewService(cat, &fakeExecProvider{})

	draft, err := svc.Create(context.Background(), "sec-fn", "main.sh", "tester")
	require.NoError(t, err)

	rotated := "name: sec-fn\nhttp:\n  auth:\n    token: brand-new-token\n"
	saved, err := svc.SaveDraft(context.Background(), "sec-fn", rotated, "main.sh", draft.Source, draft.Version)
	require.NoError(t, err)
	assert.True(t, saved.TokenSet)

	raw, err := cat.GetByName(context.Background(), "sec-fn")
	require.NoError(t, err)
	assert.Contains(t, raw.MetadataDraft, "brand-new-token")
	assert.NotContains(t, raw.MetadataDraft, "••••••••")

	cleared := "name: sec-fn\nhttp:\n  auth:\n    token: \"\"\n"
	_, err = svc.SaveDraft(context.Background(), "sec-fn", cleared, "main.sh", draft.Source, saved.Version)
	require.NoError(t, err) // empty keeps previous when previously set
	raw, err = cat.GetByName(context.Background(), "sec-fn")
	require.NoError(t, err)
	assert.Contains(t, raw.MetadataDraft, "brand-new-token")
}

func TestApplyDirValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr string
	}{
		{
			name: "missing metadata",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print(1)\n"), 0o644))
				return dir
			},
			wantErr: "metadata.yaml",
		},
		{
			name: "extra file rejected",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(validMetaYAML("extra-fn")), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print(1)\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644))
				return dir
			},
			wantErr: "unexpected file",
		},
		{
			name: "happy path publishes",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(validMetaYAML("happy-fn")), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print(1)\n"), 0o644))
				return dir
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat := newFakeCatalog()
			svc := functions.NewService(cat, &fakeExecProvider{})
			dir := tt.setup(t)
			res, err := svc.ApplyDir(context.Background(), dir, "tester")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "happy-fn", res.Name)
			assert.Equal(t, 1, res.Version)
		})
	}
}

func TestInvokeSandboxUnavailable(t *testing.T) {
	t.Parallel()
	cat := newFakeCatalog()
	seedPublished(t, cat, "no-docker", "main.py", "print(1)\n")
	svc := functions.NewService(cat, &dockerDownExecProvider{})
	svc.SetChecker(dcg.AllowAllChecker{})
	ver := 1
	_, err := svc.Invoke(context.Background(), functions.InvokeRequest{
		Name:    "no-docker",
		Version: &ver,
		Event:   map[string]any{},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnavailable)
	assert.Contains(t, err.Error(), "Docker is not running")
	assert.NotContains(t, err.Error(), "docker.sock")
}

func TestInvokeRequireVersionAndSaturation(t *testing.T) {
	t.Parallel()

	t.Run("missing version when required", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "need-ver", "main.py", "print(1)\n")
		svc := functions.NewService(cat, &fakeExecProvider{stdout: `{"ok":true}`})
		svc.SetChecker(dcg.AllowAllChecker{})
		_, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "need-ver",
			RequireVersion: true,
			Event:          map[string]any{},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidArgument)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("stdout JSON parse via invoke", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "json-fn", "main.py", "ignored")
		ver := 1
		svc := functions.NewService(cat, &fakeExecProvider{stdout: `{"sum":3}`})
		svc.SetChecker(dcg.AllowAllChecker{})
		got, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:    "json-fn",
			Version: &ver,
			Event:   map[string]any{"x": 1},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"sum": float64(3)}, got.Result)
	})

	t.Run("saturation returns rate limited", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "busy-fn", "main.py", "ignored")
		ver := 1
		started := make(chan struct{})
		release := make(chan struct{})
		svc := functions.NewServiceWithLimits(cat, &blockingExecProvider{
			started: started,
			release: release,
			stdout:  `{"ok":true}`,
		}, 1)
		svc.SetChecker(dcg.AllowAllChecker{})

		errCh := make(chan error, 1)
		go func() {
			_, err := svc.Invoke(context.Background(), functions.InvokeRequest{
				Name:    "busy-fn",
				Version: &ver,
				Event:   map[string]any{},
			})
			errCh <- err
		}()
		<-started

		_, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:    "busy-fn",
			Version: &ver,
			Event:   map[string]any{},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrRateLimited)

		close(release)
		require.NoError(t, <-errCh)
	})
}

func TestInvokeIdempotency(t *testing.T) {
	t.Parallel()

	t.Run("successful terminal replay", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "idem-ok", "main.py", "ignored")
		ver := 1
		svc := functions.NewService(cat, &fakeExecProvider{stdout: `{"n":1}`})
		svc.SetChecker(dcg.AllowAllChecker{})
		first, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "idem-ok",
			Version:        &ver,
			Event:          map[string]any{},
			IdempotencyKey: "k-ok",
		})
		require.NoError(t, err)
		require.False(t, first.Replayed)

		second, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "idem-ok",
			Version:        &ver,
			Event:          map[string]any{},
			IdempotencyKey: "k-ok",
		})
		require.NoError(t, err)
		require.True(t, second.Replayed)
		assert.Equal(t, first.RunID, second.RunID)
		assert.Equal(t, first.Result, second.Result)
		assert.Equal(t, string(types.FunctionRunSucceeded), second.Status)
	})

	t.Run("failed terminal replay keeps result body", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "idem-fail", "main.py", "ignored")
		ver := 1
		svc := functions.NewService(cat, &fakeExecProvider{stdout: "not-json", exit: 0})
		svc.SetChecker(dcg.AllowAllChecker{})
		first, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "idem-fail",
			Version:        &ver,
			Event:          map[string]any{},
			IdempotencyKey: "k-fail",
		})
		require.Error(t, err)
		require.Nil(t, first)

		second, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "idem-fail",
			Version:        &ver,
			Event:          map[string]any{},
			IdempotencyKey: "k-fail",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvokeRun)
		require.NotNil(t, second)
		assert.True(t, second.Replayed)
		assert.Equal(t, string(types.FunctionRunFailed), second.Status)
		assert.NotEmpty(t, second.Error)
	})

	t.Run("in-flight same key returns conflict not rate limited", func(t *testing.T) {
		t.Parallel()
		cat := newFakeCatalog()
		seedPublished(t, cat, "idem-busy", "main.py", "ignored")
		ver := 1
		started := make(chan struct{})
		release := make(chan struct{})
		svc := functions.NewServiceWithLimits(cat, &blockingExecProvider{
			started: started,
			release: release,
			stdout:  `{"ok":true}`,
		}, 4)
		svc.SetChecker(dcg.AllowAllChecker{})

		errCh := make(chan error, 1)
		go func() {
			_, err := svc.Invoke(context.Background(), functions.InvokeRequest{
				Name:           "idem-busy",
				Version:        &ver,
				Event:          map[string]any{},
				IdempotencyKey: "k-busy",
			})
			errCh <- err
		}()
		<-started

		got, err := svc.Invoke(context.Background(), functions.InvokeRequest{
			Name:           "idem-busy",
			Version:        &ver,
			Event:          map[string]any{},
			IdempotencyKey: "k-busy",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrConflict)
		require.Nil(t, got)
		require.NotErrorIs(t, err, types.ErrRateLimited)

		close(release)
		require.NoError(t, <-errCh)
	})
}

func validMetaYAML(name string) string {
	return "name: " + name + "\nhttp:\n  auth:\n    token: secret-token\nenv:\n  mode: test\n"
}

func seedPublished(t *testing.T, cat *fakeCatalog, name, entrypoint, source string) {
	t.Helper()
	require.NoError(t, cat.Create(context.Background(), name, validMetaYAML(name), entrypoint, source, "tester"))
	def, err := cat.GetByName(context.Background(), name)
	require.NoError(t, err)
	_, err = cat.Publish(context.Background(), name, def.Version)
	require.NoError(t, err)
}

type fakeCatalog struct {
	mu       sync.Mutex
	defs     map[string]*model.FunctionDefinition
	versions map[string]*model.FunctionDefinitionVersion
	runs     map[int64]*model.FunctionRun
	byIdem   map[string]*model.FunctionRun
	nextID   int64
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{
		defs:     map[string]*model.FunctionDefinition{},
		versions: map[string]*model.FunctionDefinitionVersion{},
		runs:     map[int64]*model.FunctionRun{},
		byIdem:   map[string]*model.FunctionRun{},
	}
}

func (c *fakeCatalog) Create(_ context.Context, name, metadata, entrypoint, source, createdBy string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.defs[name]; ok {
		return types.Errorf(types.ErrAlreadyExists, "function %q exists", name)
	}
	c.nextID++
	c.defs[name] = &model.FunctionDefinition{
		ID:              c.nextID,
		Name:            name,
		Status:          string(types.FunctionDefinitionDraft),
		Version:         1,
		CreatedBy:       createdBy,
		MetadataDraft:   metadata,
		EntrypointDraft: entrypoint,
		SourceDraft:     source,
	}
	return nil
}

func (c *fakeCatalog) GetByName(_ context.Context, name string) (*model.FunctionDefinition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *def
	return &cp, nil
}

func (c *fakeCatalog) UpdateDraft(_ context.Context, name, metadata, entrypoint, source string, version int) (*model.FunctionDefinition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	def.MetadataDraft = metadata
	def.EntrypointDraft = entrypoint
	def.SourceDraft = source
	def.Version = version + 1
	cp := *def
	return &cp, nil
}

func (c *fakeCatalog) Publish(_ context.Context, name string, version int) (*model.FunctionDefinition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	meta := def.MetadataDraft
	entry := def.EntrypointDraft
	src := def.SourceDraft
	def.MetadataPublished = &meta
	def.EntrypointPublished = &entry
	def.SourcePublished = &src
	def.Status = string(types.FunctionDefinitionPublished)
	nextPublished := 1
	for _, ver := range c.versions {
		if ver != nil && ver.FunctionName == name && ver.Version >= nextPublished {
			nextPublished = ver.Version + 1
		}
	}
	c.nextID++
	c.versions[nameVersionKey(name, nextPublished)] = &model.FunctionDefinitionVersion{
		ID:           c.nextID,
		FunctionName: name,
		Version:      nextPublished,
		Metadata:     meta,
		Entrypoint:   entry,
		Source:       src,
	}
	def.Version++
	return def, nil
}

func (c *fakeCatalog) Delete(_ context.Context, name string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.defs[name]; !ok {
		return 0, types.ErrNotFound
	}
	delete(c.defs, name)
	return 1, nil
}

func (c *fakeCatalog) ListPublished(_ context.Context) ([]*model.FunctionDefinition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*model.FunctionDefinition, 0)
	for _, def := range c.defs {
		if def.Status == string(types.FunctionDefinitionPublished) {
			cp := *def
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (c *fakeCatalog) ListAll(_ context.Context) ([]*model.FunctionDefinition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*model.FunctionDefinition, 0, len(c.defs))
	for _, def := range c.defs {
		cp := *def
		out = append(out, &cp)
	}
	return out, nil
}

func (c *fakeCatalog) GetVersion(_ context.Context, name string, version int) (*model.FunctionDefinitionVersion, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ver, ok := c.versions[nameVersionKey(name, version)]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *ver
	return &cp, nil
}

func (c *fakeCatalog) GetLatestPublished(_ context.Context, name string) (*model.FunctionDefinitionVersion, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var latest *model.FunctionDefinitionVersion
	for _, ver := range c.versions {
		if ver.FunctionName != name {
			continue
		}
		if latest == nil || ver.Version > latest.Version {
			cp := *ver
			latest = &cp
		}
	}
	if latest == nil {
		return nil, types.ErrNotFound
	}
	return latest, nil
}

func (c *fakeCatalog) CreateRun(_ context.Context, name string, version int, idempotencyKey *string) (*model.FunctionRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idempotencyKey != nil {
		if _, ok := c.byIdem[name+"|"+*idempotencyKey]; ok {
			return nil, types.Errorf(types.ErrAlreadyExists, "run exists")
		}
	}
	c.nextID++
	run := &model.FunctionRun{
		ID:           c.nextID,
		FunctionName: name,
		Version:      version,
		Status:       string(types.FunctionRunRunning),
	}
	if idempotencyKey != nil {
		k := *idempotencyKey
		run.IdempotencyKey = &k
		c.byIdem[name+"|"+k] = run
	}
	c.runs[run.ID] = run
	cp := *run
	return &cp, nil
}

func (c *fakeCatalog) GetRunByIdempotency(_ context.Context, name, key string) (*model.FunctionRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	run, ok := c.byIdem[name+"|"+key]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *run
	return &cp, nil
}

func (c *fakeCatalog) CompleteRun(_ context.Context, id int64, status string, durationMs int64, exitCode *int, errMsg string, resultJSON *string) (*model.FunctionRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	run, ok := c.runs[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	run.Status = status
	run.DurationMs = durationMs
	run.ExitCode = exitCode
	run.Error = errMsg
	run.ResultJSON = resultJSON
	cp := *run
	return &cp, nil
}

func (c *fakeCatalog) ListRuns(_ context.Context, name string) ([]*model.FunctionRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*model.FunctionRun, 0)
	for _, run := range c.runs {
		if run.FunctionName == name {
			cp := *run
			out = append(out, &cp)
		}
	}
	return out, nil
}

func nameVersionKey(name string, version int) string {
	return name + "@" + strconv.Itoa(version)
}

type dockerDownExecProvider struct{}

func (dockerDownExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{Env: dockerDownEnv{}}, nil
}

type dockerDownEnv struct {
	env.OSExecutionEnv
}

func (dockerDownEnv) Exec(_ context.Context, _ env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	cause := errors.New("failed to connect to the docker API at unix:///var/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /var/run/docker.sock: connect: no such file or directory")
	return result.Err[env.Capture, result.ExecutionError](result.NewExecutionError("spawn_error", cause.Error(), cause))
}

type fakeExecProvider struct {
	stdout string
	stderr string
	exit   int
	err    error
}

func (p *fakeExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	if p.err != nil {
		return pkgexec.Config{}, p.err
	}
	return pkgexec.Config{
		Env: &scriptedEnv{
			stdout: p.stdout,
			stderr: p.stderr,
			exit:   p.exit,
		},
	}, nil
}

type scriptedEnv struct {
	env.OSExecutionEnv
	stdout string
	stderr string
	exit   int
}

func (e *scriptedEnv) Exec(_ context.Context, _ env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	return result.Ok[env.Capture, result.ExecutionError](env.Capture{
		Stdout:   e.stdout,
		Stderr:   e.stderr,
		ExitCode: e.exit,
	})
}

type blockingExecProvider struct {
	started chan struct{}
	release chan struct{}
	stdout  string
}

func (p *blockingExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{
		Env: &blockingEnv{
			started: p.started,
			release: p.release,
			stdout:  p.stdout,
		},
	}, nil
}

type blockingEnv struct {
	env.OSExecutionEnv
	started chan struct{}
	release chan struct{}
	stdout  string
	once    sync.Once
}

func (e *blockingEnv) Exec(_ context.Context, _ env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	e.once.Do(func() { close(e.started) })
	<-e.release
	return result.Ok[env.Capture, result.ExecutionError](env.Capture{
		Stdout:   e.stdout,
		ExitCode: 0,
	})
}

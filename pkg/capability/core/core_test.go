package core_test

import (
	"context"
	"maps"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func resetCore(t *testing.T) {
	t.Helper()
	hub.Default.Unregister(hub.CapCore)
	t.Cleanup(func() { hub.Default.Unregister(hub.CapCore) })
	require.NoError(t, core.Register())
}

func TestRegisterExposesCoreOps(t *testing.T) {
	resetCore(t)
	desc, ok := hub.Default.Get(hub.CapCore)
	require.True(t, ok)
	names := map[string]struct{}{}
	for _, op := range desc.Operations {
		names[op.Name] = struct{}{}
	}
	for _, want := range []string{
		core.OpNotifySend, core.OpClipCreate, core.OpAgentRun,
		core.OpHTTPRequest, core.OpRunCode, core.OpRunTerminal,
		core.OpKVGet, core.OpKVSet, core.OpKVDelete,
	} {
		_, ok := names[want]
		assert.Truef(t, ok, "missing op %s", want)
	}
	assert.True(t, capability.IsMutation(core.OpHTTPRequest))
	assert.True(t, capability.IsMutation(core.OpNotifySend))
	assert.True(t, capability.IsMutation(core.OpKVSet))
}

func TestNormalizeAndKVRoundTrip(t *testing.T) {
	store := &memKV{data: map[string]types.KV{}}
	core.SetKVStore(store)
	t.Cleanup(func() { core.SetKVStore(nil) })
	resetCore(t)

	_, err := capability.Invoke(context.Background(), hub.CapCore, core.OpKVSet, map[string]any{
		"namespace":   "sync",
		"key":         "watermark",
		"value":       map[string]any{"n": 42},
		"ttl_seconds": 60,
	})
	require.NoError(t, err)

	res, err := capability.Invoke(context.Background(), hub.CapCore, core.OpKVGet, map[string]any{
		"namespace": "sync",
		"key":       "watermark",
	})
	require.NoError(t, err)
	val, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 42, val["n"])

	for k, v := range store.data {
		v["expires_at"] = time.Now().UTC().Add(-time.Minute).Unix()
		store.data[k] = v
	}
	_, err = capability.Invoke(context.Background(), hub.CapCore, core.OpKVGet, map[string]any{
		"namespace": "sync",
		"key":       "watermark",
	})
	assert.Error(t, err)
}

func TestHTTPRequestBlocksPrivate(t *testing.T) {
	prev := config.App.Core.HTTP
	config.App.Core.HTTP = config.CoreHTTPConfig{AllowPrivate: false}
	t.Cleanup(func() { config.App.Core.HTTP = prev })
	resetCore(t)

	_, err := capability.Invoke(context.Background(), hub.CapCore, core.OpHTTPRequest, map[string]any{
		"url": "http://127.0.0.1:9/",
	})
	assert.Error(t, err)
}

func TestHTTPRequestAllowHosts(t *testing.T) {
	prev := config.App.Core.HTTP
	config.App.Core.HTTP = config.CoreHTTPConfig{
		AllowPrivate: false,
		AllowHosts:   []string{"127.0.0.1"},
	}
	t.Cleanup(func() { config.App.Core.HTTP = prev })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	resetCore(t)
	res, err := capability.Invoke(context.Background(), hub.CapCore, core.OpHTTPRequest, map[string]any{
		"url": "http://" + ln.Addr().String() + "/",
	})
	require.NoError(t, err)
	data, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", data["body"])
}

func TestHTTPRequestBlocksRedirectToPrivate(t *testing.T) {
	prev := config.App.Core.HTTP
	t.Cleanup(func() { config.App.Core.HTTP = prev })

	redirectTo := "http://10.0.0.1:9/private"
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTo, http.StatusFound)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	config.App.Core.HTTP = config.CoreHTTPConfig{
		AllowPrivate: false,
		AllowHosts:   []string{"127.0.0.1"},
	}
	resetCore(t)
	_, err = capability.Invoke(context.Background(), hub.CapCore, core.OpHTTPRequest, map[string]any{
		"url": "http://" + ln.Addr().String() + "/",
	})
	assert.Error(t, err)
}

func TestHTTPRequestAllowPrivate(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	prev := config.App.Core.HTTP
	config.App.Core.HTTP = config.CoreHTTPConfig{AllowPrivate: true}
	t.Cleanup(func() { config.App.Core.HTTP = prev })
	resetCore(t)

	res, err := capability.Invoke(context.Background(), hub.CapCore, core.OpHTTPRequest, map[string]any{
		"url": "http://" + ln.Addr().String() + "/",
	})
	require.NoError(t, err)
	data, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 200, data["status"])
	assert.Equal(t, "ok", data["body"])
}

func TestRunTerminalViaExecProvider(t *testing.T) {
	prev := dcg.DefaultChecker()
	dcg.SetDefaultChecker(dcg.AllowAllChecker{})
	t.Cleanup(func() { dcg.SetDefaultChecker(prev) })

	root := t.TempDir()
	core.SetExecProvider(fixedExecProvider{root: root})
	t.Cleanup(func() { core.SetExecProvider(nil) })
	resetCore(t)

	res, err := capability.Invoke(context.Background(), hub.CapCore, core.OpRunTerminal, map[string]any{
		"command": "echo hello-core",
	})
	require.NoError(t, err)
	data, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data["output"], "hello-core")
}

func TestLegacyCapability(t *testing.T) {
	assert.True(t, core.LegacyCapability("notify"))
	assert.False(t, core.LegacyCapability("core"))
	op, ok := core.LegacyOperation("agent", "run")
	assert.True(t, ok)
	assert.Equal(t, core.OpAgentRun, op)
}

type fixedExecProvider struct{ root string }

func (p fixedExecProvider) ExecConfig(context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{Workspace: p.root}, nil
}

type memKV struct {
	data map[string]types.KV
}

func (*memKV) key(uid types.Uid, ns, key string) string {
	return string(uid) + "|" + ns + "|" + key
}

func (m *memKV) Get(_ context.Context, uid types.Uid, namespace, key string) (types.KV, error) {
	v, ok := m.data[m.key(uid, namespace, key)]
	if !ok {
		return nil, types.Errorf(types.ErrNotFound, "not found")
	}
	out := types.KV{}
	maps.Copy(out, v)
	return out, nil
}

func (m *memKV) Set(_ context.Context, uid types.Uid, namespace, key string, value types.KV) error {
	cp := types.KV{}
	maps.Copy(cp, value)
	m.data[m.key(uid, namespace, key)] = cp
	return nil
}

func (m *memKV) Delete(_ context.Context, uid types.Uid, namespace, key string) error {
	delete(m.data, m.key(uid, namespace, key))
	return nil
}

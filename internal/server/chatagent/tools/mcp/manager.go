// Package mcp registers outbound MCP servers as chat-agent tools.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/version"
)

var (
	defaultMu  sync.RWMutex
	defaultMgr *Manager
)

// Manager holds long-lived outbound MCP client sessions.
type Manager struct {
	mu      sync.Mutex
	servers []*boundServer
}

type boundServer struct {
	cfg     config.ChatAgentMCPServer
	session *mcpsdk.ClientSession
	tools   []remoteTool
}

// SetDefault installs the process-wide MCP client manager.
func SetDefault(m *Manager) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultMgr = m
}

// StopDefault closes the process-wide MCP client manager.
func StopDefault(ctx context.Context) error {
	defaultMu.Lock()
	mgr := defaultMgr
	defaultMgr = nil
	defaultMu.Unlock()
	if mgr == nil {
		return nil
	}
	return mgr.Stop(ctx)
}

func defaultManager() *Manager {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultMgr
}

// Start connects configured MCP servers. Failed servers are logged and skipped.
func (m *Manager) Start(ctx context.Context, servers []config.ChatAgentMCPServer, listen, apiPath string) error {
	if m == nil {
		return nil
	}
	for _, srv := range servers {
		if err := m.startOne(ctx, srv, listen, apiPath); err != nil {
			flog.Error(fmt.Errorf("chat_agent mcp_servers %s: %w", srv.Name, err))
		}
	}
	return nil
}

// Stop closes all sessions.
func (m *Manager) Stop(_ context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var errs []error
	for _, b := range m.servers {
		if b.session != nil {
			if err := b.session.Close(); err != nil {
				errs = append(errs, fmt.Errorf("mcp server %s: %w", b.cfg.Name, err))
			}
		}
		for _, t := range b.tools {
			permission.ClearMCPToolMeta(t.Name())
		}
	}
	m.servers = nil
	return errors.Join(errs...)
}

func (m *Manager) startOne(ctx context.Context, srv config.ChatAgentMCPServer, listen, apiPath string) error {
	if err := pkgmcp.ValidateServerName(strings.TrimSpace(srv.Name)); err != nil {
		return err
	}
	if strings.TrimSpace(srv.URL) != "" && pkgmcp.IsSelfMCP(srv.URL, listen, apiPath) {
		return errors.New("url must not point at this process /mcp")
	}
	warnStdioInstaller(srv)
	session, err := connect(srv)
	if err != nil {
		return err
	}
	tools, err := listRemoteTools(ctx, session)
	if err != nil {
		if closeErr := session.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	bound := &boundServer{cfg: srv, session: session}
	for _, rt := range tools {
		if rt == nil || rt.Name == "" {
			continue
		}
		if !pkgmcp.Match(srv.Include, srv.Exclude, srv.Name, rt.Name) {
			continue
		}
		readOnly, destructive := hints(rt)
		item := remoteTool{
			server: srv.Name,
			remote: rt.Name,
			desc:   rt.Description,
			schema: schemaMap(rt.InputSchema),
			call:   bound.call,
		}
		permission.SetMCPToolMeta(item.Name(), readOnly, destructive)
		bound.tools = append(bound.tools, item)
	}
	m.mu.Lock()
	m.servers = append(m.servers, bound)
	m.mu.Unlock()
	return nil
}

func (b *boundServer) call(ctx context.Context, name string, args map[string]any) (*mcpsdk.CallToolResult, error) {
	if b == nil || b.session == nil {
		return nil, fmt.Errorf("mcp server %s is not connected", b.cfg.Name)
	}
	return b.session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
}

// Register adds connected MCP tools to the agent registry.
func Register(registry *tool.Registry) error {
	mgr := defaultManager()
	if mgr == nil || registry == nil {
		return nil
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, b := range mgr.servers {
		for _, item := range b.tools {
			if err := registry.Register(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// ActiveToolNames returns registered outbound MCP tool names.
func ActiveToolNames() []string {
	mgr := defaultManager()
	if mgr == nil {
		return nil
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	var names []string
	for _, b := range mgr.servers {
		for _, item := range b.tools {
			names = append(names, item.Name())
		}
	}
	return names
}

func connect(srv config.ChatAgentMCPServer) (*mcpsdk.ClientSession, error) {
	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "flowbot-agent",
		Version: version.Buildtags,
	}, nil)
	var transport mcpsdk.Transport
	if u := strings.TrimSpace(srv.URL); u != "" {
		httpClient := &http.Client{Transport: headerRoundTripper{headers: srv.Headers, base: http.DefaultTransport}}
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint:             u,
			HTTPClient:           httpClient,
			DisableStandaloneSSE: true,
		}
	} else {
		cmd := exec.Command(srv.Command, srv.Args...) // #nosec G204 -- YAML stdio MCP, no shell
		cmd.Env = mergeEnv(os.Environ(), srv.Env)
		transport = &mcpsdk.CommandTransport{Command: cmd}
	}
	// Background: the fx OnStart context is cancelled after start and would kill the session.
	return client.Connect(context.Background(), transport, nil)
}

func warnStdioInstaller(srv config.ChatAgentMCPServer) {
	cmd := strings.TrimSpace(srv.Command)
	if cmd == "" {
		return
	}
	base := strings.ToLower(filepath.Base(cmd))
	base = strings.TrimSuffix(strings.TrimSuffix(base, ".exe"), ".cmd")
	switch base {
	case "npx", "uvx":
		flog.Warn("chat_agent.mcp_servers %s command %q fetches and runs packages at start as the Flowbot user", srv.Name, srv.Command)
	}
}

func listRemoteTools(ctx context.Context, session *mcpsdk.ClientSession) ([]*mcpsdk.Tool, error) {
	var out []*mcpsdk.Tool
	var cursor string
	for {
		params := &mcpsdk.ListToolsParams{}
		if cursor != "" {
			params.Cursor = cursor
		}
		page, err := session.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}
		if page != nil {
			out = append(out, page.Tools...)
			cursor = page.NextCursor
		}
		if cursor == "" {
			return out, nil
		}
	}
}

func hints(t *mcpsdk.Tool) (readOnly, destructive bool) {
	if t == nil || t.Annotations == nil {
		return false, false
	}
	readOnly = t.Annotations.ReadOnlyHint
	if t.Annotations.DestructiveHint != nil {
		destructive = *t.Annotations.DestructiveHint
	}
	return readOnly, destructive
}

type headerRoundTripper struct {
	headers map[string]string
	base    http.RoundTripper
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	for k, v := range h.headers {
		clone.Header.Set(k, v)
	}
	return base.RoundTrip(clone)
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	index := map[string]int{}
	out := make([]string, 0, len(base)+len(extra))
	for _, kv := range base {
		k, _, ok := strings.Cut(kv, "=")
		if ok {
			index[k] = len(out)
		}
		out = append(out, kv)
	}
	for k, v := range extra {
		entry := k + "=" + v
		if i, ok := index[k]; ok {
			out[i] = entry
			continue
		}
		out = append(out, entry)
	}
	return out
}

package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/bytedance/sonic"
	goPlugin "github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/flowline-io/flowbot/pkg/plugin"
	pb "github.com/flowline-io/flowbot/pkg/plugin/grpc/proto"
)

const (
	handshakeVersion = 1
	magicCookieKey   = "FLOWBOT_PLUGIN"
	magicCookieValue = "flowbot-plugin-v1"
)

// GrpcRunner implements plugin.Runner using hashicorp/go-plugin.
type GrpcRunner struct {
	client   *goPlugin.Client
	svc      pb.PluginServiceClient
	hostAPI  plugin.HostAPI
	manifest *plugin.Manifest
	info     *plugin.PluginInfo
	inflight sync.WaitGroup
	mu       sync.Mutex
	started  bool
}

// NewGrpcRunner creates a gRPC runner for a plugin manifest.
func NewGrpcRunner(m *plugin.Manifest) (*GrpcRunner, error) {
	cmd := exec.Command(m.GRPC.Binary, m.GRPC.Args...)
	setPdeathsig(cmd)

	return &GrpcRunner{
		manifest: m,
		client: goPlugin.NewClient(&goPlugin.ClientConfig{
			HandshakeConfig: goPlugin.HandshakeConfig{
				ProtocolVersion:  handshakeVersion,
				MagicCookieKey:   magicCookieKey,
				MagicCookieValue: magicCookieValue,
			},
			Plugins: map[string]goPlugin.Plugin{
				"module": &GrpcPlugin{},
			},
			Cmd:              cmd,
			AllowedProtocols: []goPlugin.Protocol{goPlugin.ProtocolGRPC},
			Stderr:           os.Stderr,
		}),
	}, nil
}

// SetHostAPI sets the HostAPI for this runner (called by PluginManager).
func (r *GrpcRunner) SetHostAPI(api plugin.HostAPI) {
	r.hostAPI = api
}

// Load connects to the plugin process and retrieves its PluginInfo.
func (r *GrpcRunner) Load(ctx context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	rpcClient, err := r.client.Client()
	if err != nil {
		return nil, fmt.Errorf("grpc load: connect failed: %w", err)
	}

	raw, err := rpcClient.Dispense("module")
	if err != nil {
		rpcClient.Close()
		return nil, fmt.Errorf("grpc load: dispense failed: %w", err)
	}

	svc, ok := raw.(*pb.PluginServiceClient)
	if !ok {
		return nil, fmt.Errorf("grpc load: unexpected type %T", raw)
	}
	r.svc = *svc

	r.info = &plugin.PluginInfo{
		Name:         m.Name,
		Version:      m.Version,
		Provides:     m.Provides,
		ConfigSchema: m.ConfigSchema,
	}
	return r.info, nil
}

// Start initializes the plugin via the Init() RPC.
func (r *GrpcRunner) Start(ctx context.Context, config json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.svc.Init(ctx, &pb.InitRequest{Config: string(config)})
	if err != nil {
		return fmt.Errorf("grpc start: init failed: %w", err)
	}
	r.started = true
	return nil
}

// Stop gracefully drains in-flight calls and kills the plugin process.
func (r *GrpcRunner) Stop(ctx context.Context) error {
	r.mu.Lock()
	r.started = false
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		r.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	r.client.Kill()
	return nil
}

// Call invokes a named function on the plugin.
func (r *GrpcRunner) Call(ctx context.Context, function string, params json.RawMessage) (json.RawMessage, error) {
	r.inflight.Add(1)
	defer r.inflight.Done()

	r.mu.Lock()
	started := r.started
	r.mu.Unlock()
	if !started {
		return nil, fmt.Errorf("grpc call: plugin not started")
	}

	switch function {
	case "command":
		req := &pb.CommandRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.Command(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc command: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc command: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Payload), nil

	case "form":
		req := &pb.FormRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.Form(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc form: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc form: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Payload), nil

	case "rules":
		resp, err := r.svc.Rules(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc rules: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc rules: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Rules), nil

	case "help":
		resp, err := r.svc.Help(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc help: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc help: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Help), nil

	case "bootstrap":
		_, err := r.svc.Bootstrap(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("grpc bootstrap: %w", err)
		}
		return nil, nil

	case "ability_call":
		req := &pb.CallRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.AbilityCall(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc ability_call: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc ability_call: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Result), nil

	case "webhook_convert":
		req := &pb.WebhookConvertRequest{}
		if err := sonic.Unmarshal(params, req); err != nil {
			return nil, fmt.Errorf("grpc call: unmarshal params: %w", err)
		}
		resp, err := r.svc.WebhookConvert(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc webhook_convert: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("grpc webhook_convert: plugin error: %s", resp.Error)
		}
		return json.RawMessage(resp.Result), nil

	default:
		return nil, fmt.Errorf("grpc call: unknown function %q", function)
	}
}

// Health checks the plugin's readiness.
func (r *GrpcRunner) Health(ctx context.Context) (*plugin.HealthStatus, error) {
	resp, err := r.svc.IsReady(ctx, &emptypb.Empty{})
	if err != nil {
		return &plugin.HealthStatus{Ready: false, LastError: err.Error()}, nil
	}
	return &plugin.HealthStatus{Ready: resp.Ready}, nil
}

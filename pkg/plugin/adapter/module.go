package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PluginModuleAdapter implements module.Handler by delegating to a Runner.
type PluginModuleAdapter struct {
	module.Base
	runner   atomic.Pointer[plugin.Runner]
	name     string
	manifest *plugin.Manifest
	ready    atomic.Bool
}

// NewModuleAdapter creates a module adapter for a plugin.
func NewModuleAdapter(m *plugin.Manifest, r plugin.Runner) *PluginModuleAdapter {
	a := &PluginModuleAdapter{name: m.Name, manifest: m}
	a.runner.Store(&r)
	return a
}

// SwapRunner atomically swaps the underlying runner without unregistering.
func (a *PluginModuleAdapter) SwapRunner(newRunner plugin.Runner) {
	a.runner.Store(&newRunner)
}

func (a *PluginModuleAdapter) Init(jsonconf json.RawMessage) error {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return fmt.Errorf("plugin %s: no runner", a.name)
	}
	if err := (*r).Start(context.Background(), jsonconf); err != nil {
		return err
	}
	a.ready.Store(true)
	return nil
}

func (a *PluginModuleAdapter) IsReady() bool {
	return a.ready.Load()
}

func (a *PluginModuleAdapter) Bootstrap() error {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return fmt.Errorf("plugin %s: no runner", a.name)
	}
	_, err := (*r).Call(context.Background(), "bootstrap", nil)
	return err
}

func (a *PluginModuleAdapter) Input(ctx types.Context, head types.KV, content any) (types.MsgPayload, error) {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return nil, fmt.Errorf("plugin %s: no runner", a.name)
	}
	raw, err := sonic.Marshal(map[string]any{"context": ctx, "head": head, "content": content})
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	result, err := (*r).Call(context.Background(), "input", raw)
	if err != nil {
		return nil, err
	}
	return unmarshalPayload(result)
}

func (a *PluginModuleAdapter) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return nil, fmt.Errorf("plugin %s: no runner", a.name)
	}
	raw, err := sonic.Marshal(map[string]any{"context": ctx, "content": content})
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	result, err := (*r).Call(context.Background(), "command", raw)
	if err != nil {
		return nil, err
	}
	return unmarshalPayload(result)
}

func (a *PluginModuleAdapter) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return nil, fmt.Errorf("plugin %s: no runner", a.name)
	}
	raw, err := sonic.Marshal(map[string]any{"context": ctx, "values": values})
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	result, err := (*r).Call(context.Background(), "form", raw)
	if err != nil {
		return nil, err
	}
	return unmarshalPayload(result)
}

func (a *PluginModuleAdapter) Rules() []any {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return nil
	}
	result, err := (*r).Call(context.Background(), "rules", nil)
	if err != nil {
		return nil
	}
	var rules []any
	if err := sonic.Unmarshal(result, &rules); err != nil {
		return nil
	}
	return rules
}

func (a *PluginModuleAdapter) Help() (map[string][]string, error) {
	r := a.runner.Load()
	if r == nil || *r == nil {
		return nil, fmt.Errorf("plugin %s: no runner", a.name)
	}
	result, err := (*r).Call(context.Background(), "help", nil)
	if err != nil {
		return nil, err
	}
	var help map[string][]string
	if err := sonic.Unmarshal(result, &help); err != nil {
		return nil, fmt.Errorf("unmarshal help: %w", err)
	}
	return help, nil
}

func (*PluginModuleAdapter) Webservice(_ *fiber.App) {
	// Plugin webservice routes are proxied through /service/{plugin-name}/*
}

// unmarshalPayload converts a JSON result from the plugin into a types.MsgPayload.
// It checks for a _type field to determine the concrete message type, falling back
// to TextMsg and then KVMsg.
func unmarshalPayload(data json.RawMessage) (types.MsgPayload, error) {
	var wrapper struct {
		Type string `json:"_type"`
	}
	if err := sonic.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal payload wrapper: %w", err)
	}
	if wrapper.Type != "" {
		payload := types.ToPayload(wrapper.Type, data)
		if payload != nil {
			if kv, ok := payload.(types.KVMsg); ok {
				delete(kv, "_type")
				return kv, nil
			}
			return payload, nil
		}
	}
	var tm types.TextMsg
	if err := sonic.Unmarshal(data, &tm); err == nil && tm.Text != "" {
		return tm, nil
	}
	var kv types.KV
	if err := sonic.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return types.KVMsg(kv), nil
}

package adapter

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/plugin"
)

// PluginAbilityAdapter registers Invoker closures for declared operations.
type PluginAbilityAdapter struct {
	runner     plugin.Runner
	capability hub.CapabilityType
	operations []string
}

// NewAbilityAdapter creates an ability adapter from a manifest ability declaration.
func NewAbilityAdapter(r plugin.Runner, capType string, ops []string) *PluginAbilityAdapter {
	return &PluginAbilityAdapter{
		runner:     r,
		capability: hub.CapabilityType(capType),
		operations: ops,
	}
}

// Register registers all declared operations as capability.Invoker closures.
func (a *PluginAbilityAdapter) Register() error {
	for _, op := range a.operations {
		if err := capability.RegisterInvoker(a.capability, op, a.makeInvoker(op)); err != nil {
			return fmt.Errorf("register invoker %s/%s: %w", a.capability, op, err)
		}
	}
	return nil
}

// Unregister removes all registered invokers.
func (a *PluginAbilityAdapter) Unregister() {
	for _, op := range a.operations {
		capability.UnregisterInvoker(a.capability, op)
	}
}

func (a *PluginAbilityAdapter) makeInvoker(op string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		raw, err := sonic.Marshal(struct {
			Operation string         `json:"operation"`
			Params    map[string]any `json:"params"`
		}{Operation: op, Params: params})
		if err != nil {
			return nil, fmt.Errorf("ability invoke marshal: %w", err)
		}
		result, err := a.runner.Call(ctx, "ability_call", raw)
		if err != nil {
			return nil, fmt.Errorf("ability invoke: %w", err)
		}
		var invokeResult capability.InvokeResult
		if err := sonic.Unmarshal(result, &invokeResult); err != nil {
			return nil, fmt.Errorf("ability invoke unmarshal: %w", err)
		}
		return &invokeResult, nil
	}
}

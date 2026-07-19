// Package wasm provides the WebAssembly-based plugin runner.
package wasm

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// HostBindings implements the host function imports for wasm plugins.
type HostBindings struct {
	hostAPI   plugin.HostAPI
	httpPerms map[string]bool
}

// SetAPI sets the HostAPI for the bindings.
func (h *HostBindings) SetAPI(hostAPI plugin.HostAPI) {
	h.hostAPI = hostAPI
}

// exportToRuntime registers all host functions into the wazero runtime under the "flowbot" module namespace.
func (h *HostBindings) exportToRuntime(ctx context.Context, r wazero.Runtime) error {
	b := r.NewHostModuleBuilder("flowbot")

	h.registerGetConfig(b)
	h.registerLog(b)
	h.registerKVGet(b)
	h.registerKVSet(b)
	h.registerKVDelete(b)
	h.registerHTTPRequest(b)
	h.registerEmitEvent(b)

	_, err := b.Instantiate(ctx)
	return err
}

func (h *HostBindings) registerGetConfig(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keySize, outPtr uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			keyBytes, ok := m.Memory().Read(keyPtr, keySize)
			if !ok {
				return 0
			}
			val, err := h.hostAPI.GetConfig(ctx, string(keyBytes))
			if err != nil {
				return 0
			}
			n, ok := utils.IntToUint32(len(val))
			if !ok {
				return 0
			}
			m.Memory().Write(outPtr, []byte(val))
			return n
		}).
		Export("get_config")
}

func (h *HostBindings) registerLog(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, level, msgPtr, msgSize uint32) {
			if h.hostAPI == nil {
				return
			}
			msgBytes, ok := m.Memory().Read(msgPtr, msgSize)
			if !ok {
				return
			}
			h.hostAPI.Log(ctx, levelString(level), string(msgBytes), nil)
		}).
		Export("log")
}

func (h *HostBindings) registerKVGet(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keySize, outPtr uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			keyBytes, ok := m.Memory().Read(keyPtr, keySize)
			if !ok {
				return 0
			}
			val, err := h.hostAPI.KVGet(ctx, string(keyBytes))
			if err != nil {
				return 0
			}
			n, ok := utils.IntToUint32(len(val))
			if !ok {
				return 0
			}
			m.Memory().Write(outPtr, val)
			return n
		}).
		Export("kv_get")
}

func (h *HostBindings) registerKVSet(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keySize, valPtr, valSize uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			keyBytes, ok := m.Memory().Read(keyPtr, keySize)
			if !ok {
				return 0
			}
			valBytes, ok := m.Memory().Read(valPtr, valSize)
			if !ok {
				return 0
			}
			if err := h.hostAPI.KVSet(ctx, string(keyBytes), valBytes); err != nil {
				return 0
			}
			return 1
		}).
		Export("kv_set")
}

func (h *HostBindings) registerKVDelete(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, keySize uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			keyBytes, ok := m.Memory().Read(keyPtr, keySize)
			if !ok {
				return 0
			}
			if err := h.hostAPI.KVDelete(ctx, string(keyBytes)); err != nil {
				return 0
			}
			return 1
		}).
		Export("kv_delete")
}

func (h *HostBindings) registerHTTPRequest(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, reqPtr, reqSize, outPtr, outMaxSize uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			reqBytes, ok := m.Memory().Read(reqPtr, reqSize)
			if !ok {
				return 0
			}
			var req plugin.HostHTTPRequest
			if err := sonic.Unmarshal(reqBytes, &req); err != nil {
				return 0
			}
			resp, err := h.hostAPI.HTTPRequest(ctx, &req)
			if err != nil {
				return 0
			}
			respBytes, err := sonic.Marshal(resp)
			if err != nil {
				return 0
			}
			n, ok := utils.IntToUint32(len(respBytes))
			if !ok || n > outMaxSize {
				return 0
			}
			m.Memory().Write(outPtr, respBytes)
			return n
		}).
		Export("http_request")
}

func (h *HostBindings) registerEmitEvent(b wazero.HostModuleBuilder) {
	b.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, eventPtr, eventSize uint32) uint32 {
			if h.hostAPI == nil {
				return 0
			}
			eventBytes, ok := m.Memory().Read(eventPtr, eventSize)
			if !ok {
				return 0
			}
			var event types.DataEvent
			if err := sonic.Unmarshal(eventBytes, &event); err != nil {
				return 0
			}
			if err := h.hostAPI.EmitEvent(ctx, event); err != nil {
				return 0
			}
			return 1
		}).
		Export("emit_event")
}

// levelString converts a numeric log level to its string representation.
func levelString(level uint32) string {
	switch level {
	case 0:
		return "debug"
	case 2:
		return "warn"
	case 3:
		return "error"
	default:
		return "info"
	}
}

// buildAllowlist builds a permission map from Wasm HTTP permissions.
func buildAllowlist(perms *plugin.WasmPermissions) map[string]bool {
	if perms == nil || len(perms.HTTP) == 0 {
		return nil
	}
	al := make(map[string]bool, len(perms.HTTP))
	for _, p := range perms.HTTP {
		al[p.Host] = true
	}
	return al
}

package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/tetratelabs/wazero/api"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// writeJSON writes JSON data to the wasm module's memory.
func writeJSON(ctx context.Context, mod api.Module, data any) (uint32, uint32, error) {
	raw, err := sonic.Marshal(data)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal: %w", err)
	}
	return writeBytes(ctx, mod, raw)
}

// writeBytes writes raw bytes to the wasm module's memory.
func writeBytes(ctx context.Context, mod api.Module, data []byte) (uint32, uint32, error) {
	size, ok := utils.IntToUint32(len(data))
	if !ok {
		return 0, 0, fmt.Errorf("payload too large: %d bytes", len(data))
	}
	if size == 0 {
		return 0, 0, nil
	}
	allocFn := mod.ExportedFunction("alloc")
	results, err := allocFn.Call(ctx, uint64(size))
	if err != nil {
		return 0, 0, fmt.Errorf("alloc: %w", err)
	}
	ptr, ok := utils.Uint64ToUint32(results[0])
	if !ok {
		return 0, 0, fmt.Errorf("alloc returned out-of-range pointer")
	}
	if ptr == 0 {
		return 0, 0, fmt.Errorf("alloc returned null pointer")
	}
	if !mod.Memory().Write(ptr, data) {
		return 0, 0, fmt.Errorf("memory write failed at ptr=%d size=%d", ptr, size)
	}
	return ptr, size, nil
}

// readJSON reads a JSON response from wasm memory.
// result is the raw i64 return value encoding (ptr << 32) | size.
func readJSON(_ context.Context, mod api.Module, result uint64, target any) error {
	ptr, size := decodeResult(result)
	if size == 0 {
		return nil
	}
	data, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return fmt.Errorf("memory read failed at ptr=%d size=%d", ptr, size)
	}

	// Free the buffer in wasm memory
	freeFn := mod.ExportedFunction("free")
	if freeFn != nil {
		go func() {
			if _, err := freeFn.Call(context.Background(), uint64(ptr)); err != nil {
				flog.Debug("wasm free: %v", err)
			}
		}()
	}

	// Decode JSON envelope: {"error": "...", "data": ...}
	var envelope struct {
		Error *string         `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if err := sonic.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("plugin error: %s", *envelope.Error)
	}
	if err := sonic.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}
	return nil
}

// decodeResult decodes (ptr << 32) | size
func decodeResult(result uint64) (uint32, uint32) {
	ptr := uint32(result >> 32)
	size := uint32(result & 0xFFFFFFFF)
	return ptr, size
}

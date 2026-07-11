package pipeline

import (
	"context"
	"fmt"
	"sync"
)

// DefinitionSource loads the current pipeline definitions for engine reload.
type DefinitionSource func(ctx context.Context) ([]Definition, error)

var (
	reloadMu     sync.Mutex
	reloadSource DefinitionSource
	reloadEngine *Engine
)

// SetReloadSource wires the definition loader and engine used by ReloadDefinitions.
func SetReloadSource(source DefinitionSource, engine *Engine) {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	reloadSource = source
	reloadEngine = engine
}

// ReloadDefinitions reloads the pipeline engine from the configured source.
func ReloadDefinitions(ctx context.Context) error {
	reloadMu.Lock()
	source := reloadSource
	engine := reloadEngine
	reloadMu.Unlock()

	if source == nil || engine == nil {
		return nil
	}
	defs, err := source(ctx)
	if err != nil {
		return fmt.Errorf("reload pipeline definitions: %w", err)
	}
	return engine.Reload(defs)
}

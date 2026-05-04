package template

import (
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// globalEngine is the singleton template engine used by the notification gateway.
var globalEngine struct {
	mu     sync.RWMutex
	engine *Engine
}

// Init loads templates from configuration into the global engine.
// It is safe to call multiple times (on config reload).
func Init() error {
	engine := New()
	if err := engine.LoadConfig(config.App.Notify.Templates); err != nil {
		return fmt.Errorf("failed to load notify templates: %w", err)
	}

	globalEngine.mu.Lock()
	globalEngine.engine = engine
	globalEngine.mu.Unlock()

	flog.Info("notify template engine: loaded %d templates", len(config.App.Notify.Templates))
	return nil
}

// GetEngine returns the global template engine.
func GetEngine() *Engine {
	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()
	return globalEngine.engine
}

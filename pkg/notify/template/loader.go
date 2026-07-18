package template

import (
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

// globalEngine is the singleton template engine used by the notification gateway.
var globalEngine struct {
	mu     sync.RWMutex
	engine *Engine
}

// Init loads templates into the global engine.
// It is safe to call multiple times (on reload after CRUD).
func Init(templates []manifest.Template) error {
	engine := New()
	if err := engine.LoadConfig(templates); err != nil {
		return fmt.Errorf("failed to load notify templates: %w", err)
	}

	globalEngine.mu.Lock()
	globalEngine.engine = engine
	globalEngine.mu.Unlock()

	flog.Info("notify template engine: loaded %d templates", len(templates))
	return nil
}

// GetEngine returns the global template engine.
func GetEngine() *Engine {
	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()
	return globalEngine.engine
}

// ResetForTest clears the global template engine. Intended for tests only.
func ResetForTest() {
	globalEngine.mu.Lock()
	globalEngine.engine = nil
	globalEngine.mu.Unlock()
}

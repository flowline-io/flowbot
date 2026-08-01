package pipeline

import "sync"

var (
	activeMu      sync.Mutex
	activeEngine  *Engine
	activeService *Service
)

// SetActiveEngine wires the package-level Engine used by ActiveEngine and ExecuteManual callers.
func SetActiveEngine(engine *Engine) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeEngine = engine
}

// ActiveEngine returns the wired Engine, or nil if not set.
func ActiveEngine() *Engine {
	activeMu.Lock()
	defer activeMu.Unlock()
	return activeEngine
}

// SetActiveService wires the package-level Service used by HTTP modules.
func SetActiveService(svc *Service) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeService = svc
}

// ActiveService returns the wired Service, or nil if not set.
func ActiveService() *Service {
	activeMu.Lock()
	defer activeMu.Unlock()
	return activeService
}

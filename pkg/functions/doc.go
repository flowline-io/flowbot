// Package functions is the deep module for named function apply and invoke.
package functions

import "sync"

var (
	activeMu      sync.Mutex
	activeService *Service
)

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

package homelab

import (
	"sort"
	"sync"
)

type Registry struct {
	mu   sync.RWMutex
	apps map[string]App
}

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{apps: make(map[string]App)}
}

func (r *Registry) Replace(apps []App) {
	next := make(map[string]App, len(apps))
	for _, app := range apps {
		next[app.Name] = app
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.apps = next
}

func (r *Registry) List() []App {
	r.mu.RLock()
	defer r.mu.RUnlock()
	apps := make([]App, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
	return apps
}

func (r *Registry) Get(name string) (App, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[name]
	return app, ok
}

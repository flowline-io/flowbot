package homelab

import (
	"cmp"
	"slices"
	"sync"
)

type Registry struct {
	mu          sync.RWMutex
	apps        map[string]App
	permissions Permissions
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
	slices.SortFunc(apps, func(a, b App) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return apps
}

func (r *Registry) Get(name string) (App, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[name]
	return app, ok
}

func (r *Registry) SetPermissions(p Permissions) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.permissions = p
}

func (r *Registry) Permissions() Permissions {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.permissions
}

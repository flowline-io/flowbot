package homelab

import (
	"cmp"
	"slices"
	"sync"
)

// Registry holds the current set of discovered homelab applications
// and their associated permissions, protected by a read-write mutex.
type Registry struct {
	mu          sync.RWMutex
	apps        map[string]App
	permissions Permissions
}

// DefaultRegistry is the process-wide registry instance.
var DefaultRegistry = NewRegistry()

var (
	rescanMu sync.RWMutex
	rescanFn func() error
)

// SetRunRescan registers the function used to trigger a full homelab rescan.
// Safe for concurrent use.
func SetRunRescan(fn func() error) {
	rescanMu.Lock()
	defer rescanMu.Unlock()
	rescanFn = fn
}

// LoadRunRescan returns the currently registered rescan function, or nil.
func LoadRunRescan() func() error {
	rescanMu.RLock()
	defer rescanMu.RUnlock()
	return rescanFn
}

// RunRescan triggers a full homelab scan + probe + registry update.
// Returns nil if no rescan function has been registered.
func RunRescan() error {
	fn := LoadRunRescan()
	if fn == nil {
		return nil
	}
	return fn()
}

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

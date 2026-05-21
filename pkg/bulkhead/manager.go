package bulkhead

import (
	"runtime"
	"sync"
	"time"
)

// manager holds named Bulkhead instances created lazily via Get.
type manager struct {
	mu        sync.Mutex
	instances map[string]*Bulkhead
	defaults  config
}

var defaultManager = &manager{instances: make(map[string]*Bulkhead)}

func init() {
	n := runtime.GOMAXPROCS(0)
	defaultManager.defaults = config{
		maxConcurrent: n * 4,
		maxQueue:      n * 4,
		timeout:       30 * time.Second,
	}
}

// Get returns the Bulkhead for the given name, creating one with default config if needed.
func Get(name string) *Bulkhead {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	if b, ok := defaultManager.instances[name]; ok {
		return b
	}

	cfg := defaultManager.defaults
	b := New(name,
		WithMaxConcurrent(cfg.maxConcurrent),
		WithMaxQueue(cfg.maxQueue),
		WithTimeout(cfg.timeout),
		WithOnEnter(cfg.onEnter),
		WithOnLeave(cfg.onLeave),
		WithOnDrop(cfg.onDrop),
	)
	defaultManager.instances[name] = b
	return b
}

// SetDefaults sets default options applied to all Bulkhead instances created via Get.
// Must be called before any Get calls for the settings to take effect.
func SetDefaults(opts ...Option) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	for _, o := range opts {
		o(&defaultManager.defaults)
	}
}

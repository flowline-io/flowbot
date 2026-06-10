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
	extraOpts []Option
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

	opts := []Option{
		WithMaxConcurrent(defaultManager.defaults.maxConcurrent),
		WithMaxQueue(defaultManager.defaults.maxQueue),
		WithTimeout(defaultManager.defaults.timeout),
	}
	if defaultManager.defaults.onEnter != nil {
		opts = append(opts, WithOnEnter(defaultManager.defaults.onEnter))
	}
	if defaultManager.defaults.onLeave != nil {
		opts = append(opts, WithOnLeave(defaultManager.defaults.onLeave))
	}
	if defaultManager.defaults.onDrop != nil {
		opts = append(opts, WithOnDrop(defaultManager.defaults.onDrop))
	}
	if defaultManager.defaults.onQueueEnter != nil {
		opts = append(opts, WithOnQueueEnter(defaultManager.defaults.onQueueEnter))
	}
	if defaultManager.defaults.onQueueLeave != nil {
		opts = append(opts, WithOnQueueLeave(defaultManager.defaults.onQueueLeave))
	}
	opts = append(opts, defaultManager.extraOpts...)
	b := New(name, opts...)
	defaultManager.instances[name] = b
	return b
}

// SetDefaults sets default options applied to all Bulkhead instances created via Get.
// Must be called before any Get calls for the settings to take effect.
func SetDefaults(opts ...Option) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	defaultManager.extraOpts = append(defaultManager.extraOpts, opts...)
	for _, o := range opts {
		o(&defaultManager.defaults)
	}
}

// Reset clears all cached Bulkhead instances and extra default options.
// Intended for use in tests to isolate state between cases.
func Reset() {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	defaultManager.instances = make(map[string]*Bulkhead)
	defaultManager.extraOpts = nil
}

package capability

import (
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
)

// eventPoolConfig holds config values read at InitEventPool time.
type eventPoolConfig struct {
	size    int
	expiry  time.Duration
	metrics *metrics.CapabilityCollector
}

// eventPool wraps an ants.PoolWithFunc for nonblocking event emission.
type eventPool struct {
	pool   *ants.PoolWithFunc
	config eventPoolConfig
}

// eventTask bundles the data needed by the pool worker function.
type eventTask struct {
	capability string
	operation  string
	fn         func()
}

var (
	epMu   sync.Mutex
	epInst *eventPool
)

// InitEventPool creates the global event pool. Must be called once during startup.
// Call ShutdownEventPool during server shutdown.
func InitEventPool(size int, expiryDuration string, mc *metrics.CapabilityCollector) error {
	epMu.Lock()
	defer epMu.Unlock()

	if epInst != nil {
		flog.Warn("ability: event pool already initialized")
		return nil
	}

	expiry, err := time.ParseDuration(expiryDuration)
	if err != nil {
		expiry = 30 * time.Second
	}

	cfg := eventPoolConfig{
		size:    size,
		expiry:  expiry,
		metrics: mc,
	}

	pool, err := ants.NewPoolWithFunc(size, func(i any) {
		task, ok := i.(*eventTask)
		if !ok {
			flog.Warn("ability: event pool received unknown task type: %T", i)
			return
		}
		task.fn()
	}, ants.WithNonblocking(true), ants.WithExpiryDuration(expiry),
		ants.WithPanicHandler(func(i any) {
			flog.Warn("ability: event emitter panicked: %v", i)
		}),
	)
	if err != nil {
		return err
	}

	epInst = &eventPool{pool: pool, config: cfg}
	flog.Info("ability: event pool initialized (size=%d, expiry=%s)", pool.Cap(), expiry)
	return nil
}

// ShutdownEventPool releases the pool, waiting up to 30s for in-flight tasks.
func ShutdownEventPool() {
	epMu.Lock()
	ep := epInst
	epInst = nil
	epMu.Unlock()

	if ep == nil {
		return
	}
	ep.pool.ReleaseTimeout(30 * time.Second)
	flog.Info("ability: event pool released")
}

// GetEventPool returns the global event pool, or nil if not initialized.
func GetEventPool() *ants.PoolWithFunc {
	epMu.Lock()
	defer epMu.Unlock()
	if epInst == nil {
		return nil
	}
	return epInst.pool
}

// submitEvent submits an event emission function to the pool.
// Falls back to direct execution if pool is nil (not initialized).
func submitEvent(capability, operation string, fn func()) {
	epMu.Lock()
	ep := epInst
	epMu.Unlock()

	if ep == nil {
		fn()
		return
	}

	task := &eventTask{
		capability: capability,
		operation:  operation,
		fn:         fn,
	}

	err := ep.pool.Invoke(task)
	if err != nil {
		if err == ants.ErrPoolClosed {
			fn()
			return
		}
		reason := "unknown"
		if err == ants.ErrPoolOverload {
			reason = "pool_overload"
		}
		flog.Warn("ability(%s.%s): event dropped: %v", capability, operation, err)
		if ep.config.metrics != nil {
			ep.config.metrics.IncEventDropped(capability, operation, reason)
		}
	}
}

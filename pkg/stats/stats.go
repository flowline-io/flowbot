// Logic related to expvar handling: reporting live stats such as
// session and topic counts, memory usage etc.
// The stats updates happen in a separate go routine to avoid
// locking on main logic routines.

package stats

import (
	"encoding/json"
	"expvar"
	"runtime"
	"sort"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// StatsUpdate Runtime statistics communication channel.
var update chan *varUpdate

// A simple implementation of histogram expvar.Var.
// `Bounds` specifies the histogram buckets as follows (length = len(bounds)):
//
//	(-inf, Bounds[i]) for i = 0
//	[Bounds[i-1], Bounds[i]) for 0 < i < length
//	[Bounds[i-1], +inf) for i = length
type histogram struct {
	Count          int64     `json:"count"`
	Sum            float64   `json:"sum"`
	CountPerBucket []int64   `json:"count_per_bucket"`
	Bounds         []float64 `json:"bounds"`
}

func (h *histogram) addSample(v float64) {
	h.Count++
	h.Sum += v
	idx := sort.SearchFloat64s(h.Bounds, v)
	h.CountPerBucket[idx]++
}

func (h *histogram) String() string {
	if r, err := json.Marshal(h); err == nil {
		return string(r)
	}
	return ""
}

type varUpdate struct {
	// Name of the variable to update
	varname string
	// Value to publish (int, float, etc.)
	value any
	// Treat the count as an increment as opposite to the final value.
	inc bool
}

// Init Initialize stats reporting through expvar.
func Init(app *fiber.App, path string) {
	if path == "" || path == "-" {
		return
	}

	app.Get(path, adaptor.HTTPHandler(expvar.Handler()))
	update = make(chan *varUpdate, 1024)

	start := time.Now()
	expvar.Publish("Uptime", expvar.Func(func() any {
		return time.Since(start).Seconds()
	}))
	expvar.Publish("NumGoroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	go updater()

	flog.Info("stats: variables exposed at '%s'", path)
}

func RegisterDbStats() {
	if f := store.Store.DbStats(); f != nil {
		expvar.Publish("DbStats", expvar.Func(f))
	}
}

// RegisterInt Register integer variable. Don't check for initialization.
func RegisterInt(name string) {
	expvar.Publish(name, new(expvar.Int))
}

// RegisterHistogram Register histogram variable. `bounds` specifies histogram buckets/bins
// (see comment next to the `histogram` struct definition).
func RegisterHistogram(name string, bounds []float64) {
	numBuckets := len(bounds) + 1
	expvar.Publish(name, &histogram{
		CountPerBucket: make([]int64, numBuckets),
		Bounds:         bounds,
	})
}

// Set Async publish int variable.
func Set(name string, val int64) {
	if update != nil {
		select {
		case update <- &varUpdate{name, val, false}:
		default:
		}
	}
}

// Inc Async publish an increment (decrement) to int variable.
func Inc(name string, val int) {
	if update != nil {
		select {
		case update <- &varUpdate{name, int64(val), true}:
		default:
		}
	}
}

// AddHistSample Async publish a value (add a sample) to a histogram variable.
func AddHistSample(name string, val float64) {
	if update != nil {
		select {
		case update <- &varUpdate{varname: name, value: val}:
		default:
		}
	}
}

// Shutdown Stop publishing stats.
func Shutdown() {
	if update != nil {
		update <- nil
	}
}

// The go routine which actually publishes stats updates.
func updater() {
	for upd := range update {
		if upd == nil {
			update = nil
			// Don't care to close the channel.
			break
		}

		// Handle var update
		if ev := expvar.Get(upd.varname); ev != nil {
			switch v := ev.(type) {
			case *expvar.Int:
				count := upd.value.(int64)
				if upd.inc {
					v.Add(count)
				} else {
					v.Set(count)
				}
			case *histogram:
				val := upd.value.(float64)
				v.addSample(val)
			default:
				flog.Panic("stats: unsupported expvar type %T", ev)
			}
		} else {
			panic("stats: update to unknown variable " + upd.varname)
		}
	}

	flog.Info("stats: shutdown")
}

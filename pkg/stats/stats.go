// Package stats provides runtime statistics and metrics collection.
package stats

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	BotTotalStatsName                     = "bot_total"
	BotRunTotalStatsName                  = "bot_run_total"
	ModuleTotalStatsName                  = "module_total"
	ModuleRunTotalStatsName               = "module_run_total"
	BookmarkTotalStatsName                = "bookmark_total"
	SearchTotalStatsName                  = "search_total"
	SearchProcessedDocumentTotalStatsName = "search_processed_document_total"
	QueueProcessedTasksTotalStatsName     = "queue_processed_tasks_total"
	QueueFailedTasksTotalStatsName        = "queue_failed_tasks_total"
	QueueInProgressTasksStatsName         = "queue_in_progress_tasks"
	EventTotalStatsName                   = "event_total"
	TorrentDownloadTotalStatsName         = "torrent_download_total"
	TorrentStatusTotalStatsName           = "torrent_status_total"
	GiteaIssueTotalStatsName              = "gitea_issue_total"
	KanbanEventTotalStatsName             = "kanban_event_total"
	KanbanTaskTotalStatsName              = "kanban_task_total"
	ReaderTotalStatsName                  = "reader_total"
	ReaderUnreadTotalStatsName            = "reader_unread_total"
	MonitorUpTotalStatsName               = "monitor_up_total"
	MonitorDownTotalStatsName             = "monitor_down_total"
	DockerContainerTotalStatsName         = "docker_container_total"
)

type RulesetLabel string

const (
	InputRuleset   RulesetLabel = "input"
	AgentRuleset   RulesetLabel = "agent"
	CommandRuleset RulesetLabel = "command"
	CronRuleset    RulesetLabel = "cron"
	FormRuleset    RulesetLabel = "form"
)

var (
	// global registry
	registry = prometheus.NewRegistry()

	// pushgateway configuration
	pushGatewayURL = "http://localhost:9091" // default pushgateway address
	jobName        = "flowbot"
	pushInterval   = 15 * time.Second

	// maps for counters and gauges
	counters = make(map[string]prometheus.Counter)
	gauges   = make(map[string]prometheus.Gauge)
	mu       sync.RWMutex

	// pusher
	pusher *push.Pusher

	// initialize once
	once sync.Once
)

var initialized atomic.Bool

var (
	statsInstance     *Stats
	statsInstanceOnce sync.Once
)

// Stats provides access to the global Prometheus registry for creating vector metrics.
type Stats struct {
	vecCounters map[string]*prometheus.CounterVec
	vecGauges   map[string]*prometheus.GaugeVec
	vecHistos   map[string]*prometheus.HistogramVec
}

// NewStats creates a Stats wrapper around the global Prometheus registry.
// Returns a singleton instance. Returns nil when metrics has not been initialized (metrics.enabled=false).
func NewStats() *Stats {
	if !initialized.Load() {
		return nil
	}
	statsInstanceOnce.Do(func() {
		statsInstance = &Stats{
			vecCounters: make(map[string]*prometheus.CounterVec),
			vecGauges:   make(map[string]*prometheus.GaugeVec),
			vecHistos:   make(map[string]*prometheus.HistogramVec),
		}
	})
	return statsInstance
}

// MetricsConfig configuration struct
type MetricsConfig struct {
	PushGatewayURL string
	JobName        string
	PushInterval   time.Duration
}

// Init initializes the metrics system
func Init(config *MetricsConfig) error {
	hostid, hostname, err := utils.HostInfo()
	if err != nil {
		return fmt.Errorf("failed to get host info: %w", err)
	}

	once.Do(func() {
		initialized.Store(true)
		if config != nil {
			if config.PushGatewayURL != "" {
				pushGatewayURL = config.PushGatewayURL
			}
			if config.JobName != "" {
				jobName = config.JobName
			}
			if config.PushInterval > 0 {
				pushInterval = config.PushInterval
			}
		}

		// create pusher
		pusher = push.New(pushGatewayURL, jobName).Gatherer(registry)
		pusher.Grouping("instance", hostid)
		pusher.Grouping("hostname", hostname)

		// start periodic pushing
		go func() {
			ticker := time.NewTicker(pushInterval)
			defer ticker.Stop()

			for range ticker.C {
				if err := pusher.Push(); err != nil {
					log.Printf("stats: failed to push metrics: %v", err)
				}
			}
		}()
	})
	return nil
}

// getOrCreateCounter gets or creates a counter
func getOrCreateCounter(name string, labels prometheus.Labels) prometheus.Counter {
	mu.Lock()
	defer mu.Unlock()

	// create a unique key
	key := fmt.Sprintf("counter_%s_%v", name, labels)

	if counter, exists := counters[key]; exists {
		return counter
	}

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        name + "_counter", // add suffix to avoid conflicts
		Help:        fmt.Sprintf("Counter for %s", name),
		ConstLabels: labels,
	})

	registry.MustRegister(counter)
	counters[key] = counter

	return counter
}

// getOrCreateGauge gets or creates a gauge
func getOrCreateGauge(name string, labels prometheus.Labels) prometheus.Gauge {
	mu.Lock()
	defer mu.Unlock()

	// create a unique key
	key := fmt.Sprintf("gauge_%s_%v", name, labels)

	if gauge, exists := gauges[key]; exists {
		return gauge
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name + "_gauge", // add suffix to avoid conflicts
		Help:        fmt.Sprintf("Gauge for %s", name),
		ConstLabels: labels,
	})

	registry.MustRegister(gauge)
	gauges[key] = gauge

	return gauge
}

// PushNow immediately pushes metrics to pushgateway
func PushNow() error {
	if pusher == nil {
		return fmt.Errorf("metrics not initialized, call Init() first")
	}
	return pusher.Push()
}

// PushWithContext pushes metrics using a context
func PushWithContext(ctx context.Context) error {
	if pusher == nil {
		return fmt.Errorf("metrics not initialized, call Init() first")
	}
	return pusher.PushContext(ctx)
}

// RegisterCounterVec creates or returns an existing CounterVec registered with the global registry.
func (s *Stats) RegisterCounterVec(name, help string, labelNames ...string) *prometheus.CounterVec {
	if s == nil {
		panic("stats: RegisterCounterVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if cv, exists := s.vecCounters[name]; exists {
		return cv
	}

	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labelNames)
	registry.MustRegister(cv)
	s.vecCounters[name] = cv
	return cv
}

// RegisterGaugeVec creates or returns an existing GaugeVec registered with the global registry.
func (s *Stats) RegisterGaugeVec(name, help string, labelNames ...string) *prometheus.GaugeVec {
	if s == nil {
		panic("stats: RegisterGaugeVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if gv, exists := s.vecGauges[name]; exists {
		return gv
	}

	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labelNames)
	registry.MustRegister(gv)
	s.vecGauges[name] = gv
	return gv
}

// RegisterHistogramVec creates or returns an existing HistogramVec registered with the global registry.
func (s *Stats) RegisterHistogramVec(name, help string, labelNames ...string) *prometheus.HistogramVec {
	if s == nil {
		panic("stats: RegisterHistogramVec called on nil Stats")
	}
	mu.Lock()
	defer mu.Unlock()

	if hv, exists := s.vecHistos[name]; exists {
		return hv
	}

	hv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.DefBuckets,
	}, labelNames)
	registry.MustRegister(hv)
	s.vecHistos[name] = hv
	return hv
}

// SetInitializedForTesting sets the initialized flag for test purposes.
// Returns the previous value.
func SetInitializedForTesting(val bool) bool {
	return initialized.Swap(val)
}

// MetricInterface compatibility interface supporting common methods for Counter and Gauge
type MetricInterface interface {
	Inc()
	Add(float64)
	Set(uint64) // compatibility with older code
}

// metricWrapper wrapper supporting both counter and gauge operations
type metricWrapper struct {
	counter prometheus.Counter
	gauge   prometheus.Gauge
}

func (m *metricWrapper) Inc() {
	if m.counter != nil {
		m.counter.Inc()
	}
	if m.gauge != nil {
		m.gauge.Inc()
	}
}

func (m *metricWrapper) Add(val float64) {
	if m.counter != nil {
		m.counter.Add(val)
	}
	if m.gauge != nil {
		m.gauge.Add(val)
	}
}

func (m *metricWrapper) Set(val uint64) {
	if m.gauge != nil {
		m.gauge.Set(float64(val))
	}
}

// getOrCreateMetric gets or creates a metric with both counter and gauge
func getOrCreateMetric(name string, labels prometheus.Labels) MetricInterface {
	counter := getOrCreateCounter(name, labels)
	gauge := getOrCreateGauge(name, labels)

	return &metricWrapper{
		counter: counter,
		gauge:   gauge,
	}
}

func BotTotalCounter() MetricInterface {
	return getOrCreateMetric(BotTotalStatsName, prometheus.Labels{})
}

// ModuleTotalCounter is an alias for BotTotalCounter using the new module naming.
func ModuleTotalCounter() MetricInterface {
	return getOrCreateMetric(ModuleTotalStatsName, prometheus.Labels{})
}

func BotRunTotalCounter(rulesetLabel RulesetLabel) MetricInterface {
	return getOrCreateMetric(BotRunTotalStatsName, prometheus.Labels{
		"ruleset": string(rulesetLabel),
	})
}

// ModuleRunTotalCounter is an alias for BotRunTotalCounter using the new module naming.
func ModuleRunTotalCounter(rulesetLabel RulesetLabel) MetricInterface {
	return getOrCreateMetric(ModuleRunTotalStatsName, prometheus.Labels{
		"ruleset": string(rulesetLabel),
	})
}

func BookmarkTotalCounter() MetricInterface {
	return getOrCreateMetric(BookmarkTotalStatsName, prometheus.Labels{})
}

func SearchTotalCounter(index string) MetricInterface {
	return getOrCreateMetric(SearchTotalStatsName, prometheus.Labels{
		"index": index,
	})
}

func SearchProcessedDocumentTotalCounter(index string) MetricInterface {
	return getOrCreateMetric(SearchProcessedDocumentTotalStatsName, prometheus.Labels{
		"index": index,
	})
}

func QueueProcessedTasksTotalCounter(taskType string) MetricInterface {
	return getOrCreateMetric(QueueProcessedTasksTotalStatsName, prometheus.Labels{
		"task_type": taskType,
	})
}

func QueueFailedTasksTotalCounter(taskType string) MetricInterface {
	return getOrCreateMetric(QueueFailedTasksTotalStatsName, prometheus.Labels{
		"task_type": taskType,
	})
}

func QueueInProgressTasksCounter(taskType string) MetricInterface {
	return getOrCreateMetric(QueueInProgressTasksStatsName, prometheus.Labels{
		"task_type": taskType,
	})
}

func EventTotalCounter() MetricInterface {
	return getOrCreateMetric(EventTotalStatsName, prometheus.Labels{})
}

func TorrentDownloadTotalCounter() MetricInterface {
	return getOrCreateMetric(TorrentDownloadTotalStatsName, prometheus.Labels{})
}

func TorrentStatusTotalCounter(status string) MetricInterface {
	return getOrCreateMetric(TorrentStatusTotalStatsName, prometheus.Labels{
		"status": status,
	})
}

func GiteaIssueTotalCounter(status string) MetricInterface {
	return getOrCreateMetric(GiteaIssueTotalStatsName, prometheus.Labels{
		"status": status,
	})
}

func KanbanEventTotalCounter(name string) MetricInterface {
	return getOrCreateMetric(KanbanEventTotalStatsName, prometheus.Labels{
		"event_name": name,
	})
}

func KanbanTaskTotalCounter() MetricInterface {
	return getOrCreateMetric(KanbanTaskTotalStatsName, prometheus.Labels{})
}

func ReaderTotalCounter() MetricInterface {
	return getOrCreateMetric(ReaderTotalStatsName, prometheus.Labels{})
}

func ReaderUnreadTotalCounter() MetricInterface {
	return getOrCreateMetric(ReaderUnreadTotalStatsName, prometheus.Labels{})
}

func DockerContainerTotalCounter() MetricInterface {
	return getOrCreateMetric(DockerContainerTotalStatsName, prometheus.Labels{})
}

func MonitorUpTotalCounter() MetricInterface {
	return getOrCreateMetric(MonitorUpTotalStatsName, prometheus.Labels{})
}

func MonitorDownTotalCounter() MetricInterface {
	return getOrCreateMetric(MonitorDownTotalStatsName, prometheus.Labels{})
}

const (
	CacheHitTotalStatsName      = "cache_hit_total"
	CacheMissTotalStatsName     = "cache_miss_total"
	CacheEvictionTotalStatsName = "cache_eviction_total"
	CacheSizeBytesStatsName     = "cache_size_bytes"
)

// CacheHitTotalCounter returns a metric for tracking cache hit count by backend.
func CacheHitTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheHitTotalStatsName, prometheus.Labels{"backend": backend})
}

// CacheMissTotalCounter returns a metric for tracking cache miss count by backend.
func CacheMissTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheMissTotalStatsName, prometheus.Labels{"backend": backend})
}

// CacheEvictionTotalCounter returns a metric for tracking cache eviction count by backend.
func CacheEvictionTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheEvictionTotalStatsName, prometheus.Labels{"backend": backend})
}

// CacheSizeBytesGauge returns a metric for tracking approximate cache memory usage by backend.
func CacheSizeBytesGauge(backend string) MetricInterface {
	return getOrCreateMetric(CacheSizeBytesStatsName, prometheus.Labels{"backend": backend})
}

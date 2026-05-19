package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsSystem(t *testing.T) {
	t.Run("metrics initialization and counters", func(t *testing.T) {
		config := &MetricsConfig{
			PushGatewayURL: "http://localhost:9091",
			JobName:        "flowbot-test",
			PushInterval:   1 * time.Second,
		}

		err := Init(config)
		require.NoError(t, err, "Failed to initialize stats")

		botCounter := BotTotalCounter()
		botCounter.Inc()
		botCounter.Add(5)
		botCounter.Set(10)

		botRunCounter := BotRunTotalCounter(AgentRuleset)
		botRunCounter.Inc()

		searchCounter := SearchTotalCounter("test-index")
		searchCounter.Inc()

		queueCounter := QueueProcessedTasksTotalCounter("test-task")
		queueCounter.Add(3)

		BookmarkTotalCounter().Set(100)
		EventTotalCounter().Inc()
		TorrentDownloadTotalCounter().Set(50)
		TorrentStatusTotalCounter("downloading").Set(5)
		GiteaIssueTotalCounter("open").Set(20)
		KanbanEventTotalCounter("task_created").Inc()
		KanbanTaskTotalCounter().Set(30)
		ReaderTotalCounter().Set(200)
		ReaderUnreadTotalCounter().Set(15)
		DockerContainerTotalCounter().Set(8)
		MonitorUpTotalCounter().Set(5)
		MonitorDownTotalCounter().Set(2)

		CacheHitTotalCounter("redis").Inc()
		CacheMissTotalCounter("redis").Inc()
		CacheEvictionTotalCounter("redis").Inc()
		CacheSizeBytesGauge("redis").Set(1024)

		err = PushNow()
		if err != nil {
			t.Logf("Push failed (expected if no pushgateway): %v", err)
		}

		t.Log("Stats system test completed successfully")
	})
}

func TestMetricInterface(t *testing.T) {
	t.Run("counter inc add and gauge set operations", func(t *testing.T) {
		metric := BotTotalCounter()

		metric.Inc()
		metric.Add(5.5)
		metric.Set(100)

		t.Log("MetricInterface test completed successfully")
	})
}

func TestRegisterVecMetrics(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, s *Stats)
	}{
		{
			name: "counter vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				cv := s.RegisterCounterVec("test_counter_vec_total", "help", "label_a")
				require.NotNil(t, cv)
				cv.WithLabelValues("val1").Inc()
				cv.WithLabelValues("val1").Inc()
				cv.WithLabelValues("val2").Inc()
			},
		},
		{
			name: "gauge vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				gv := s.RegisterGaugeVec("test_gauge_vec", "help", "label_a")
				require.NotNil(t, gv)
				gv.WithLabelValues("val1").Set(42)
				gv.WithLabelValues("val2").Inc()
				gv.WithLabelValues("val2").Dec()
			},
		},
		{
			name: "histogram vec registers and works",
			fn: func(t *testing.T, s *Stats) {
				hv := s.RegisterHistogramVec("test_histogram_vec_seconds", "help", "label_a")
				require.NotNil(t, hv)
				hv.WithLabelValues("val1").Observe(1.5)
				hv.WithLabelValues("val2").Observe(0.5)
			},
		},
		{
			name: "duplicate registration returns same vec",
			fn: func(t *testing.T, s *Stats) {
				cv1 := s.RegisterCounterVec("test_dup_vec_total", "help", "l")
				cv2 := s.RegisterCounterVec("test_dup_vec_total", "help", "l")
				assert.Same(t, cv1, cv2)
			},
		},
		{
			name: "nil stats panics on register",
			fn: func(t *testing.T, s *Stats) {
				assert.Panics(t, func() {
					var nilStats *Stats
					nilStats.RegisterCounterVec("x", "h", "l")
				})
			},
		},
	}
	// Set initialized once before all subtests since they share global state.
	old := SetInitializedForTesting(true)
	defer SetInitializedForTesting(old)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStats()
			tt.fn(t, s)
		})
	}
}

func TestNewStatsWhenNotInitialized(t *testing.T) {
	t.Run("returns nil when Init not called", func(t *testing.T) {
		old := SetInitializedForTesting(false)
		defer SetInitializedForTesting(old)
		assert.Nil(t, NewStats())
	})
}

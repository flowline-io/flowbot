package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatsSystem(t *testing.T) {
	t.Parallel()
	t.Run("metrics initialization and counters", func(t *testing.T) {
		t.Parallel()
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
	t.Parallel()
	t.Run("counter inc add and gauge set operations", func(t *testing.T) {
		t.Parallel()
		metric := BotTotalCounter()

		metric.Inc()
		metric.Add(5.5)
		metric.Set(100)

		t.Log("MetricInterface test completed successfully")
	})
}

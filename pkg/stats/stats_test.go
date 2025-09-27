package stats

import (
	"testing"
	"time"
)

func TestStatsSystem(t *testing.T) {
	// Test initialization
	config := &MetricsConfig{
		PushGatewayURL: "http://localhost:9091",
		JobName:        "flowbot-test",
		PushInterval:   1 * time.Second, // Short interval used for testing
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Failed to initialize stats: %v", err)
	}

	// Test basic metrics
	botCounter := BotTotalCounter()
	botCounter.Inc()
	botCounter.Add(5)
	botCounter.Set(10)

	// Test metrics with labels
	botRunCounter := BotRunTotalCounter(AgentRuleset)
	botRunCounter.Inc()

	searchCounter := SearchTotalCounter("test-index")
	searchCounter.Inc()

	queueCounter := QueueProcessedTasksTotalCounter("test-task")
	queueCounter.Add(3)

	// Test all other metrics
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

	// Test immediate push (if pushgateway is available)
	// Note: this test may fail if pushgateway is not running
	err = PushNow()
	if err != nil {
		t.Logf("Push failed (expected if no pushgateway): %v", err)
	}

	t.Log("Stats system test completed successfully")
}

func TestMetricInterface(t *testing.T) {
	// Test all methods of the interface
	metric := BotTotalCounter()

	// Test Inc method
	metric.Inc()

	// Test Add method
	metric.Add(5.5)

	// Test Set method
	metric.Set(100)

	t.Log("MetricInterface test completed successfully")
}

package uptimekuma

// MonitorStatusMetric Monitor Status (1 = UP, 0= DOWN, 2= PENDING, 3= MAINTENANCE)
const MonitorStatusMetric = "monitor_status"

const (
	UP          = 1
	DOWN        = 0
	PENDING     = 2
	MAINTENANCE = 3
)

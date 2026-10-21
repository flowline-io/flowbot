package web

import analyticsmod "github.com/flowline-io/flowbot/internal/modules/analytics"

var analyticsSvc *analyticsmod.Service

// SetAnalyticsService installs the analytics domain service used by web handlers.
func SetAnalyticsService(s *analyticsmod.Service) {
	analyticsSvc = s
}

func analyticsService() *analyticsmod.Service {
	return analyticsSvc
}

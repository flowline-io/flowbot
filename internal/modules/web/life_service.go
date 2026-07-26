package web

import (
	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
)

var webLifeService *lifemod.Service

// SetLifeService installs the Life domain service used by web handlers.
func SetLifeService(s *lifemod.Service) {
	webLifeService = s
}

func lifeService() *lifemod.Service {
	if webLifeService != nil {
		return webLifeService
	}
	return lifemod.ActiveService()
}

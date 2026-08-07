package server

import (
	"strings"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	capgw "github.com/flowline-io/flowbot/pkg/capability/gateway"
	"github.com/flowline-io/flowbot/pkg/config"
)

func initGatewayAbility() error {
	if !config.App.Gateway.Enabled {
		return nil
	}
	applyGatewayPermissionDefault()
	if storepkg.Database != nil && storepkg.Database.GetClient() != nil {
		capgw.SetJobStore(storepkg.GatewayStoreFromDB())
	}
	return capgw.Register()
}

func applyGatewayPermissionDefault() {
	switch strings.ToLower(strings.TrimSpace(config.App.Gateway.Permission)) {
	case "allow":
		permission.SetGatewayDefaultAction(permission.ActionAllow)
	default:
		permission.SetGatewayDefaultAction(permission.ActionAsk)
	}
}

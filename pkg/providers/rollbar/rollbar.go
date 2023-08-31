package rollbar

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/rollbar/rollbar-go"
)

const (
	ID = "rollbar"

	EnableKey      = "enable"
	TokenKey       = "token"
	EnvironmentKey = "environment"
	ServerRootKey  = "server_root"
)

func Setup() error {
	enableVal, err := providers.GetConfig(ID, EnableKey)
	if err != nil {
		return err
	}
	if !enableVal.Bool() {
		flog.Info("rollbar disable")
		return nil
	}

	tokenVal, err := providers.GetConfig(ID, TokenKey)
	if err != nil {
		return err
	}
	envVal, err := providers.GetConfig(ID, EnvironmentKey)
	if err != nil {
		return err
	}
	rootVal, err := providers.GetConfig(ID, ServerRootKey)
	if err != nil {
		return err
	}
	rollbar.SetToken(tokenVal.String())
	rollbar.SetEnvironment(envVal.String())
	rollbar.SetServerRoot(rootVal.String())
	return nil
}

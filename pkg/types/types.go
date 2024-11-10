package types

import (
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/lithammer/shortuuid/v4"
)

func Id() string {
	return shortuuid.New()
}

func AppUrl() string {
	return config.App.Flowbot.URL
}

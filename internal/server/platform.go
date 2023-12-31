package server

import (
	"github.com/flowline-io/flowbot/internal/platforms/discord"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
)

func hookPlatform(stop <-chan bool) {
	// slack
	go slack.NewDriver().WebSocketClient(stop)
	// discord
	go discord.HandleWebsocket(stop)
}

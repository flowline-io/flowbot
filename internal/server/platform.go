package server

import (
	"github.com/flowline-io/flowbot/internal/platforms/discord"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
)

func hookPlatform(stop <-chan bool) {
	// slack
	d := slack.Driver{}
	go d.WebSocketClient(stop)
	// discord
	go discord.HandleWebsocket(stop)
}
